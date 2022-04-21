package concurrent

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

var _ ApplicationCC = (*TimedApplication)(nil)

type TimedApplication struct {
	checkTxSpan      time.Duration
	deliverTxSpan    time.Duration
	preCheckTxSpan   time.Duration
	preDeliverTxSpan time.Duration
	querySpan        time.Duration
	recheckSpan      time.Duration
	checkingTx       bool
	deliveringTx     bool
}

func (app *TimedApplication) Info(types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{}
}
func (app *TimedApplication) SetOption(types.RequestSetOption) types.ResponseSetOption {
	return types.ResponseSetOption{}
}
func (app *TimedApplication) Query(types.RequestQuery) types.ResponseQuery {
	time.Sleep(app.querySpan)
	return types.ResponseQuery{}
}

// Mempool Connection
func (app *TimedApplication) CheckTx(tx types.RequestCheckTx) types.ResponseCheckTx {
	app.checkingTx = true
	fmt.Println("Start CheckTx")
	time.Sleep(app.checkTxSpan)
	fmt.Println("Finish CheckTx")
	app.checkingTx = false
	return types.ResponseCheckTx{}
}
func (app *TimedApplication) ReCheckTx(tx types.RequestCheckTx) types.ResponseCheckTx {
	time.Sleep(app.recheckSpan)
	return types.ResponseCheckTx{}
}

// Consensus Connection
func (app *TimedApplication) InitChain(types.RequestInitChain) types.ResponseInitChain {
	return types.ResponseInitChain{}
}
func (app *TimedApplication) BeginBlock(types.RequestBeginBlock) types.ResponseBeginBlock {
	return types.ResponseBeginBlock{}
}
func (app *TimedApplication) DeliverTx(tx types.RequestDeliverTx) types.ResponseDeliverTx {
	app.deliveringTx = true
	fmt.Println("Start DeliverTx")
	time.Sleep(app.deliverTxSpan)
	fmt.Println("Stop DeliverTx")
	app.deliveringTx = false
	return types.ResponseDeliverTx{}
}
func (app *TimedApplication) EndBlock(types.RequestEndBlock) types.ResponseEndBlock {
	return types.ResponseEndBlock{}
}
func (app *TimedApplication) Commit() types.ResponseCommit {
	if app.checkingTx || app.deliveringTx {
		panic("Commit cannot be called when delivering or checking txs")
	}
	return types.ResponseCommit{}
}

func (app *TimedApplication) PreCheckTx(tx types.RequestCheckTx) types.ResponseCheckTx {
	fmt.Println("Start PreCheckTx")
	time.Sleep(app.preCheckTxSpan)
	fmt.Println("Stop PreCheckTx")
	return types.ResponseCheckTx{}
}
func (app *TimedApplication) PreDeliverTx(tx types.RequestDeliverTx) types.ResponseDeliverTx {
	fmt.Println("Start PreDeliverTx")
	time.Sleep(app.preDeliverTxSpan)
	fmt.Println("Stop PreDeliverTx")
	return types.ResponseDeliverTx{}
}

func (cli *TimedApplication) StartRecovery(manifest *types.Manifest) error {
	return nil
}
func (cli *TimedApplication) WriteRecoveryChunk(hash types.SHA256Sum, chunk *types.AppStateChunk, isComplete bool) error {
	return nil
}

var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "TestLogger")

func TestNewAsyncLocalClient(t *testing.T) {
	assert := assert.New(t)
	app := &TimedApplication{}
	app.checkTxSpan = time.Millisecond * 50
	app.preCheckTxSpan = time.Millisecond * 50
	app.deliverTxSpan = time.Millisecond * 50
	app.preDeliverTxSpan = time.Millisecond * 50

	cli := NewAsyncLocalClient(app, logger, new(sync.RWMutex),
		new(sync.WaitGroup), new(sync.Mutex), new(sync.Mutex), new(sync.Mutex))
	cli.Start()
	cli.SetResponseCallback(func(*types.Request, *types.Response) {})
	assert.NotNil(cli, "Failed to create AsyncLocalClient")
	tx := make([]byte, 8)
	// if all are sequential, it needs 300ms
	expectStop := time.Now().Add(time.Millisecond * 300)
	nonExpectShort := time.Now().Add(time.Millisecond * 25)
	for i := 0; i < 2; i++ {
		go cli.CheckTxAsync(types.RequestCheckTx{Tx:tx})
		go cli.DeliverTxAsync(types.RequestDeliverTx{Tx:tx})
	}
	time.Sleep(time.Millisecond * 5) //wait for go routine to start.
	cli.CommitAsync()
	assert.True(time.Now().After(nonExpectShort), "Run too quick")
	assert.True(time.Now().Before(expectStop), "Run too slow")
	cli.Stop()
}

func TestReadAPI(t *testing.T) {
	assert := assert.New(t)
	app := &TimedApplication{}
	app.checkTxSpan = time.Millisecond * 50
	app.preCheckTxSpan = time.Millisecond * 50
	app.deliverTxSpan = time.Millisecond * 50
	app.preDeliverTxSpan = time.Millisecond * 50
	app.querySpan = time.Millisecond * 50
	cli := NewAsyncLocalClient(app, logger, new(sync.RWMutex),
		new(sync.WaitGroup), new(sync.Mutex), new(sync.Mutex), new(sync.Mutex))
	cli.Start()
	cli.SetResponseCallback(func(*types.Request, *types.Response) {})
	assert.NotNil(cli, "Failed to create AsyncLocalClient")
	tx := make([]byte, 8)
	expectStop := time.Now().Add(time.Millisecond * 300)
	nonExpectShort := time.Now().Add(time.Millisecond * 25)
	reqQuery := types.RequestQuery{}
	reqInfo := types.RequestInfo{}
	for i := 0; i < 2; i++ {
		go cli.CheckTxAsync(types.RequestCheckTx{Tx:tx})
		go cli.DeliverTxAsync(types.RequestDeliverTx{Tx:tx})
		for j := 0; j < 10; j++ {
			go cli.QueryAsync(reqQuery)
			go cli.InfoAsync(reqInfo)
		}
	}
	time.Sleep(time.Millisecond * 5) //wait for go routine to start.
	cli.CommitAsync()
	assert.True(time.Now().After(nonExpectShort), "Run too quick")
	assert.True(time.Now().Before(expectStop), "Run too slow")
	cli.Stop()
}
