package concurrent

import (
	"sync"

	"github.com/cosmos/cosmos-sdk/server/concurrent/pool"

	"github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
)

var _ abcicli.Client = (*asyncLocalClient)(nil)

// asyncLocalClient is a variant from local_client.
// It makes ABCI calling more complex:
// 1. CheckTx/DeliverTx/Query/Info can be called concurrently
// 2. Other API would block calling CheckTx/DeliverTx/Query

const (
	WorkerPoolSize  = 16
	WorkerPoolSpawn = 4
	WorkerPoolQueue = 16
)

type WorkItem struct {
	reqRes *abcicli.ReqRes
	mtx    *sync.Mutex // make sure the eventual execution sequence
}

type localAsyncClientCreator struct {
	app types.Application
	log log.Logger

	commitLock     *sync.Mutex
	checkTxLowLock *sync.Mutex
	checkTxMidLock *sync.Mutex
	wgCommit       *sync.WaitGroup
	rwLock         *sync.RWMutex
}

type asyncLocalClient struct {
	cmn.BaseService
	Application ApplicationCC
	abcicli.Callback

	checkTxPool   *pool.Pool
	deliverTxPool *pool.Pool

	commitLock     *sync.Mutex
	checkTxLowLock *sync.Mutex
	checkTxMidLock *sync.Mutex
	wgCommit       *sync.WaitGroup
	rwLock         *sync.RWMutex

	checkTxQueue   chan WorkItem
	deliverTxQueue chan WorkItem
	log            log.Logger
}

func NewAsyncLocalClient(app types.Application, log log.Logger,
	rwLock *sync.RWMutex, wgCommit *sync.WaitGroup,
	commitLock, checkTxLowLock, checkTxMidLock *sync.Mutex) *asyncLocalClient {
	appcc, ok := app.(ApplicationCC)
	if !ok {
		return nil
	}
	cli := &asyncLocalClient{
		Application:    appcc,
		checkTxPool:    pool.NewPool(WorkerPoolSize/2, WorkerPoolQueue/2, WorkerPoolSpawn/2),
		deliverTxPool:  pool.NewPool(WorkerPoolSize, WorkerPoolQueue, WorkerPoolSpawn),
		checkTxQueue:   make(chan WorkItem, WorkerPoolQueue*2),
		deliverTxQueue: make(chan WorkItem, WorkerPoolQueue*2),
		log:            log,
		commitLock:     commitLock,
		checkTxLowLock: checkTxLowLock,
		checkTxMidLock: checkTxMidLock,
		wgCommit:       wgCommit,
		rwLock:         rwLock,
	}
	cli.BaseService = *cmn.NewBaseService(nil, "asyncLocalClient", cli)
	return cli
}

func (app *asyncLocalClient) OnStart() error {
	if err := app.BaseService.OnStart(); err != nil {
		return err
	}
	go app.checkTxWorker()
	go app.deliverTxWorker()
	return nil
}

func (app *asyncLocalClient) OnStop() {
	app.BaseService.OnStop()
	app.commitLock.Lock()
	close(app.checkTxQueue)
	close(app.deliverTxQueue)
	app.commitLock.Unlock()
}

func (app *asyncLocalClient) SetResponseCallback(cb abcicli.Callback) {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	app.Callback = cb
}

func (app *asyncLocalClient) checkTxWorker() {
	for i := range app.checkTxQueue {
		i.mtx.Lock() // wait the PreCheckTx finish
		i.mtx.Unlock()
		func() {
			app.rwLock.Lock()         // make sure not other non-CheckTx/non-DeliverTx ABCI is called
			defer app.rwLock.Unlock() // this unlock is put after wgCommit.Done() to give commit priority
			if i.reqRes.Response == nil {
				tx := types.RequestCheckTx{Tx: i.reqRes.Request.GetCheckTx().GetTx()}
				res := app.Application.CheckTx(tx)
				i.reqRes.Response = types.ToResponseCheckTx(res) // Set response
			}
			i.reqRes.Done()
			app.wgCommit.Done() // enable Commit to start
			if cb := i.reqRes.GetCallback(); cb != nil {
				cb(i.reqRes.Response)
			}
			app.Callback(i.reqRes.Request, i.reqRes.Response)
		}()
	}
}

func (app *asyncLocalClient) deliverTxWorker() {
	for i := range app.deliverTxQueue {
		i.mtx.Lock() // wait the PreDeliverTx finish
		i.mtx.Unlock()
		func() {
			app.rwLock.Lock()         // make sure not other non-CheckTx/non-DeliverTx ABCI is called
			defer app.rwLock.Unlock() // this unlock is put after wgCommit.Done() to give commit priority
			if i.reqRes.Response == nil {
				tx := types.RequestDeliverTx{Tx: i.reqRes.Request.GetDeliverTx().GetTx()}
				res := app.Application.DeliverTx(tx)
				i.reqRes.Response = types.ToResponseDeliverTx(res) // Set response
			}
			i.reqRes.Done()
			app.wgCommit.Done() // enable Commit to start
			if cb := i.reqRes.GetCallback(); cb != nil {
				cb(i.reqRes.Response)
			}
			app.Callback(i.reqRes.Request, i.reqRes.Response)
		}()
	}
}

// TODO: change types.Application to include Error()?
func (app *asyncLocalClient) Error() error {
	return nil
}

func (app *asyncLocalClient) FlushAsync() *abcicli.ReqRes {
	// Do nothing
	return newLocalReqRes(types.ToRequestFlush(), nil)
}

func (app *asyncLocalClient) EchoAsync(msg string) *abcicli.ReqRes {
	return app.callback(
		types.ToRequestEcho(msg),
		types.ToResponseEcho(msg),
	)
}

func (app *asyncLocalClient) InfoAsync(req types.RequestInfo) *abcicli.ReqRes {
	app.rwLock.RLock()
	res := app.Application.Info(req)
	app.rwLock.RUnlock()
	return app.callback(
		types.ToRequestInfo(req),
		types.ToResponseInfo(res),
	)
}

func (app *asyncLocalClient) SetOptionAsync(req types.RequestSetOption) *abcicli.ReqRes {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	res := app.Application.SetOption(req)
	return app.callback(
		types.ToRequestSetOption(req),
		types.ToResponseSetOption(res),
	)
}

func (app *asyncLocalClient) DeliverTxAsync(req types.RequestDeliverTx) *abcicli.ReqRes {
	// no app level lock because the real DeliverTx would be called in the worker routine
	reqp := types.ToRequestDeliverTx(req)
	reqres := abcicli.NewReqRes(reqp)
	mtx := new(sync.Mutex)
	mtx.Lock()
	app.deliverTxQueue <- WorkItem{reqRes: reqres, mtx: mtx}
	//no need to lock commitLock because Commit and DeliverTx will not be called concurrently
	app.wgCommit.Add(1)
	app.deliverTxPool.Schedule(func() {
		defer mtx.Unlock()
		res := app.Application.PreDeliverTx(req)
		if !res.IsOK() { // no need to call the real DeliverTx
			reqres.Response = types.ToResponseDeliverTx(res)
		}
	})

	return reqres
}

func (app *asyncLocalClient) CheckTxAsync(req types.RequestCheckTx) *abcicli.ReqRes {
	// no app level lock because the real CheckTx would be called in the worker routine
	reqp := types.ToRequestCheckTx(req)
	reqres := abcicli.NewReqRes(reqp)
	mtx := new(sync.Mutex)
	mtx.Lock()
	app.checkTxLowLock.Lock()
	app.checkTxMidLock.Lock()
	app.commitLock.Lock() // here would block further queue if commit is ready to go
	app.checkTxMidLock.Unlock()
	app.checkTxQueue <- WorkItem{reqRes: reqres, mtx: mtx}
	app.wgCommit.Add(1)
	app.commitLock.Unlock()
	app.checkTxLowLock.Unlock()
	app.checkTxPool.Schedule(func() {
		defer mtx.Unlock()
		res := app.Application.PreCheckTx(req)
		if !res.IsOK() { // no need to call the real CheckTx
			reqres.Response = types.ToResponseCheckTx(res)
		}
	})
	return reqres
}

//ReCheckTxAsync here still runs synchronously
func (app *asyncLocalClient) ReCheckTxAsync(req types.RequestCheckTx) *abcicli.ReqRes {
	app.rwLock.Lock() // wont
	defer app.rwLock.Unlock()
	res := app.Application.ReCheckTx(req)
	return app.callback(
		types.ToRequestCheckTx(req),
		types.ToResponseCheckTx(res),
	)
}

// QueryAsync is supposed to run concurrently when there is no CheckTx/DeliverTx/Commit
func (app *asyncLocalClient) QueryAsync(req types.RequestQuery) *abcicli.ReqRes {
	app.rwLock.RLock()
	res := app.Application.Query(req)
	app.rwLock.RUnlock()
	return app.callback(
		types.ToRequestQuery(req),
		types.ToResponseQuery(res),
	)
}

func (app *asyncLocalClient) CommitAsync() *abcicli.ReqRes {
	app.log.Debug("Trying to get CommitAsync lock")
	app.checkTxMidLock.Lock()
	app.commitLock.Lock() // this must come before the wgCommit.Wait()
	defer app.commitLock.Unlock()
	app.checkTxMidLock.Unlock()
	app.wgCommit.Wait() // wait for all the submitted CheckTx/DeliverTx/Query finish
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	// only checkTxLock is locked here
	// because we trust deliver and commit will not call concurrently
	app.log.Debug("Start CommitAsync")
	res := app.Application.Commit()
	app.log.Debug("Finish CommitAsync")
	return app.callback(
		types.ToRequestCommit(),
		types.ToResponseCommit(res),
	)
}

func (app *asyncLocalClient) InitChainAsync(req types.RequestInitChain) *abcicli.ReqRes {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	res := app.Application.InitChain(req)
	reqRes := app.callback(
		types.ToRequestInitChain(req),
		types.ToResponseInitChain(res),
	)
	return reqRes
}

func (app *asyncLocalClient) BeginBlockAsync(req types.RequestBeginBlock) *abcicli.ReqRes {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	res := app.Application.BeginBlock(req)
	return app.callback(
		types.ToRequestBeginBlock(req),
		types.ToResponseBeginBlock(res),
	)
}

func (app *asyncLocalClient) EndBlockAsync(req types.RequestEndBlock) *abcicli.ReqRes {
	app.log.Debug("Trying to get EndBlockAsync lock")
	app.checkTxMidLock.Lock()
	app.commitLock.Lock() // this must come before the wgCommit.Wait()
	defer app.commitLock.Unlock()
	app.checkTxMidLock.Unlock()
	app.wgCommit.Wait() // wait for all the submitted CheckTx/DeliverTx/Query finish
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	// only checkTxLock is locked here
	// because we trust deliver and commit will not call concurrently
	app.log.Debug("Starting EndBlockAsync")
	res := app.Application.EndBlock(req)
	app.log.Debug("Finish EndBlockAsync")
	return app.callback(
		types.ToRequestEndBlock(req),
		types.ToResponseEndBlock(res),
	)
}

//-------------------------------------------------------

func (app *asyncLocalClient) FlushSync() error {
	return nil
}

func (app *asyncLocalClient) EchoSync(msg string) (*types.ResponseEcho, error) {
	return &types.ResponseEcho{Message: msg}, nil
}

func (app *asyncLocalClient) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	app.rwLock.RLock()
	res := app.Application.Info(req)
	app.rwLock.RUnlock()
	return &res, nil
}

func (app *asyncLocalClient) SetOptionSync(req types.RequestSetOption) (*types.ResponseSetOption, error) {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	res := app.Application.SetOption(req)
	return &res, nil
}

func (app *asyncLocalClient) DeliverTxSync(req types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	app.log.Debug("Start DeliverTxSync")
	res := app.Application.DeliverTx(req)
	return &res, nil
}

func (app *asyncLocalClient) CheckTxSync(req types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	app.log.Debug("Start CheckTxSync")
	res := app.Application.CheckTx(req)
	return &res, nil
}

func (app *asyncLocalClient) QuerySync(req types.RequestQuery) (*types.ResponseQuery, error) {
	app.rwLock.RLock()
	res := app.Application.Query(req)
	app.rwLock.RUnlock()
	return &res, nil
}

func (app *asyncLocalClient) CommitSync() (*types.ResponseCommit, error) {
	app.log.Debug("Trying to get CommitSync Lock")
	app.checkTxMidLock.Lock()
	app.commitLock.Lock() // this must come before the wgCommit.Wait()
	defer app.commitLock.Unlock()
	app.checkTxMidLock.Unlock()
	app.wgCommit.Wait() // wait for all the submitted CheckTx/DeliverTx/Query finish
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	// only checkTxLock is locked here
	// because we trust deliver and commit will not call concurrently
	app.log.Debug("Start CommitSync")
	res := app.Application.Commit()
	app.log.Debug("Finish CommitSync")
	return &res, nil
}

func (app *asyncLocalClient) InitChainSync(req types.RequestInitChain) (*types.ResponseInitChain, error) {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	res := app.Application.InitChain(req)
	return &res, nil
}

func (app *asyncLocalClient) BeginBlockSync(req types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	res := app.Application.BeginBlock(req)
	return &res, nil
}

func (app *asyncLocalClient) EndBlockSync(req types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	app.log.Debug("Trying to get EndBlockSync lock")
	app.checkTxMidLock.Lock()
	app.commitLock.Lock() // this must come before the wgCommit.Wait()
	defer app.commitLock.Unlock()
	app.checkTxMidLock.Unlock()
	app.wgCommit.Wait() // wait for all the submitted CheckTx/DeliverTx/Query finish
	app.rwLock.Lock()
	defer app.rwLock.Unlock()
	app.log.Debug("Start EndBlockSync")
	// only checkTxLock is locked here
	// because we trust deliver and commit will not call concurrently
	res := app.Application.EndBlock(req)
	app.log.Debug("Finish EndBlockSync")
	return &res, nil
}

//-------------------------------------------------------

func (app *asyncLocalClient) StartRecovery(manifest *types.Manifest) error {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()

	return app.Application.StartRecovery(manifest)
}

func (app *asyncLocalClient) WriteRecoveryChunk(hash types.SHA256Sum, chunk *types.AppStateChunk, isComplete bool) error {
	app.rwLock.Lock()
	defer app.rwLock.Unlock()

	return app.Application.WriteRecoveryChunk(hash, chunk, isComplete)
}

//-------------------------------------------------------

func (app *asyncLocalClient) callback(req *types.Request, res *types.Response) *abcicli.ReqRes {
	app.Callback(req, res)
	return newLocalReqRes(req, res)
}

func newLocalReqRes(req *types.Request, res *types.Response) *abcicli.ReqRes {
	reqRes := abcicli.NewReqRes(req)
	reqRes.Response = res
	reqRes.SetDone()
	return reqRes
}

func NewAsyncLocalClientCreator(app types.Application, log log.Logger) proxy.ClientCreator {
	return &localAsyncClientCreator{
		app:            app,
		log:            log,
		rwLock:         new(sync.RWMutex),
		wgCommit:       new(sync.WaitGroup),
		commitLock:     new(sync.Mutex),
		checkTxLowLock: new(sync.Mutex),
		checkTxMidLock: new(sync.Mutex),
	}
}

func (l *localAsyncClientCreator) NewABCIClient() (abcicli.Client, error) {
	return NewAsyncLocalClient(l.app, l.log, l.rwLock, l.wgCommit,
		l.commitLock, l.checkTxLowLock, l.checkTxMidLock), nil
}
