package baseapp

import (
	"fmt"
	"io"
	"runtime/debug"
	"strconv"
	"strings"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/snapshot"
	tmstore "github.com/tendermint/tendermint/store"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// Key to store the header in the DB itself.
// Use the db directly instead of a store to avoid
// conflicts with handlers writing to the store
// and to avoid affecting the Merkle root.
var dbHeaderKey = []byte("header")

const (
	// we pass txHash of current handling message via context so that we can publish it as metadata of Msg
	TxHashKey = "txHash"
	// we pass txSrc of current handling message via context so that we can publish it as metadata of Msg
	TxSourceKey = "txSrc"
	//this number should be around the size of the transactions in a block, TODO: configurable
	TxMsgCacheSize = 4000
)

// BaseApp reflects the ABCI application implementation.
type BaseApp struct {
	// initialized on creation
	Logger      log.Logger
	name        string               // application name from abci.Info
	db          dbm.DB               // common DB backend
	cms         sdk.CommitMultiStore // Main (uncached) state
	router      Router               // handle any kind of message
	queryRouter QueryRouter          // router for redirecting query calls
	codespacer  *sdk.Codespacer      // handle module codespacing
	collect     sdk.CollectConfig

	TxDecoder sdk.TxDecoder // unmarshal []byte into sdk.Tx

	anteHandler sdk.AnteHandler // ante handler for fee and auth
	preChecker  sdk.PreChecker

	// may be nil
	initChainer      sdk.InitChainer  // initialize state with validators and state blob
	beginBlocker     sdk.BeginBlocker // logic to run before any txs
	endBlocker       sdk.EndBlocker   // logic to run after all txs, and to determine valset changes
	addrPeerFilter   sdk.PeerFilter   // filter peers by address and port
	pubkeyPeerFilter sdk.PeerFilter   // filter peers by public key

	//--------------------
	// Volatile
	// CheckState is set on initialization and reset on Commit.
	// DeliverState is set in InitChain and BeginBlock and cleared on Commit.
	// See methods SetCheckState and SetDeliverState.
	CheckState   *state // for CheckTx
	DeliverState *state // for DeliverTx

	AccountStoreCache sdk.AccountStoreCache
	txMsgCache        *lru.Cache
	Pool              *sdk.Pool

	// Snapshot for state sync related fields
	StateSyncHelper *store.StateSyncHelper // manage state sync related status

	// flag for sealing
	sealed bool
}

var _ abci.Application = (*BaseApp)(nil)

// NewBaseApp returns a reference to an initialized BaseApp.
//
// TODO: Determine how to use a flexible and robust configuration paradigm that
// allows for sensible defaults while being highly configurable
// (e.g. functional options).
//
// NOTE: The db is used to store the version number for now.
// Accepts a user-defined TxDecoder
// Accepts variable number of option functions, which act on the BaseApp to set configuration choices
func NewBaseApp(name string, logger log.Logger, db dbm.DB, txDecoder sdk.TxDecoder, collectConfig sdk.CollectConfig, options ...func(*BaseApp)) *BaseApp {
	cache, err := lru.New(TxMsgCacheSize)
	if err != nil {
		panic(err)
	}
	app := &BaseApp{
		Logger:      logger,
		name:        name,
		db:          db,
		cms:         store.NewCommitMultiStore(db),
		router:      NewRouter(),
		queryRouter: NewQueryRouter(),
		codespacer:  sdk.NewCodespacer(),
		TxDecoder:   txDecoder,
		collect:     collectConfig,
		txMsgCache:  cache,
		Pool:        new(sdk.Pool),
	}

	sdk.UpgradeMgr.AddConfig(sdk.MainNetConfig) // TODO: make this configurable

	// Register the undefined & root codespaces, which should not be used by
	// any modules.
	app.codespacer.RegisterOrPanic(sdk.CodespaceRoot)
	for _, option := range options {
		option(app)
	}
	return app
}

// BaseApp Name
func (app *BaseApp) Name() string {
	return app.name
}

// SetCommitMultiStoreTracer sets the store tracer on the BaseApp's underlying
// CommitMultiStore.
func (app *BaseApp) SetCommitMultiStoreTracer(w io.Writer) {
	app.cms.WithTracer(w)
}

// Register the next available codespace through the baseapp's codespacer, starting from a default
func (app *BaseApp) RegisterCodespace(codespace sdk.CodespaceType) sdk.CodespaceType {
	return app.codespacer.RegisterNext(codespace)
}

// Mount IAVL stores to the provided keys in the BaseApp multistore
func (app *BaseApp) MountStoresIAVL(keys ...*sdk.KVStoreKey) {
	for _, key := range keys {
		app.MountStore(key, sdk.StoreTypeIAVL)
	}
}

// Mount stores to the provided keys in the BaseApp multistore
func (app *BaseApp) MountStoresTransient(keys ...*sdk.TransientStoreKey) {
	for _, key := range keys {
		app.MountStore(key, sdk.StoreTypeTransient)
	}
}

// Mount a store to the provided key in the BaseApp multistore, using a specified DB
func (app *BaseApp) MountStoreWithDB(key sdk.StoreKey, typ sdk.StoreType, db dbm.DB) {
	app.cms.MountStoreWithDB(key, typ, db)
}

// Mount a store to the provided key in the BaseApp multistore, using the default DB
func (app *BaseApp) MountStore(key sdk.StoreKey, typ sdk.StoreType) {
	app.cms.MountStoreWithDB(key, typ, nil)
}

// only load latest multi store application version
func (app *BaseApp) LoadCMSLatestVersion() error {
	err := app.cms.LoadLatestVersion()
	if err != nil {
		return err
	}
	return nil
}

// load latest application version
func (app *BaseApp) LoadLatestVersion(mainKey sdk.StoreKey) error {
	err := app.cms.LoadLatestVersion()
	if err != nil {
		return err
	}
	return app.initFromStore(mainKey)
}

// InitFromStore initializes the remaining logic from app.cms
func (app *BaseApp) InitFromStore(mainKey sdk.StoreKey) error {
	return app.initFromStore(mainKey)
}

// load application version
func (app *BaseApp) LoadVersion(version int64, mainKey sdk.StoreKey) error {
	err := app.cms.LoadVersion(version)
	if err != nil {
		return err
	}
	return app.initFromStore(mainKey)
}

// the last CommitID of the multistore
func (app *BaseApp) LastCommitID() sdk.CommitID {
	return app.cms.LastCommitID()
}

// the last committed block height
func (app *BaseApp) LastBlockHeight() int64 {
	return app.cms.LastCommitID().Version
}

//
func (app *BaseApp) GetCommitMultiStore() sdk.CommitMultiStore {
	return app.cms
}

func LoadBlockDB() dbm.DB {
	return LoadDB("blockstore")
}

func LoadStateDB() dbm.DB {
	return LoadDB("state")
}

func LoadTxDB() dbm.DB {
	return LoadDB("tx_index")
}

func LoadDB(dbName string) dbm.DB {
	conf := cfg.DefaultConfig()
	err := viper.Unmarshal(conf)
	if err != nil {
		panic(err)
	}

	dbType := dbm.DBBackendType(conf.DBBackend)
	return dbm.NewDB(dbName, dbType, conf.DBDir())
}

// initializes the remaining logic from app.cms
func (app *BaseApp) initFromStore(mainKey sdk.StoreKey) error {
	// main store should exist.
	// TODO: we don't actually need the main store here
	main := app.cms.GetKVStore(mainKey)
	if main == nil {
		return errors.New("baseapp expects MultiStore with 'main' KVStore")
	}
	// Needed for `gaiad export`, which inits from store but never calls initchain
	appHeight := app.LastBlockHeight()
	if appHeight == 0 {
		app.SetCheckState(abci.Header{})
	} else {
		blockDB := LoadBlockDB()
		blockStore := tmstore.NewBlockStore(blockDB)
		// note here we use appHeight, not current block store height, appHeight may be far behind storeHeight
		lastHeader := blockStore.LoadBlock(appHeight).Header
		app.SetCheckState(tmtypes.TM2PB.Header(&lastHeader))
		blockDB.Close()
	}

	//TODO(#118): figure out what does this mean! If we keep this, we will get panic: Router() on sealed BaseApp at github.com/BiJie/BinanceChain/app.(*BinanceChain).GetRouter(0xc0004bc080, 0xc000c14000, 0xc0007b9808)
	//        /Users/zhaocong/go/src/github.com/BiJie/BinanceChain/app/app.go:297 +0x6b
	//app.Seal()

	return nil
}

// NewContext returns a new Context with the correct store, the given header, and nil txBytes.
func (app *BaseApp) NewContext(mode sdk.RunTxMode, header abci.Header) sdk.Context {
	var ms sdk.CacheMultiStore
	var accountCache sdk.AccountCache

	switch mode {
	case sdk.RunTxModeDeliver:
		ms = app.DeliverState.ms
		accountCache = app.DeliverState.AccountCache
	default:
		ms = app.CheckState.ms
		accountCache = app.CheckState.AccountCache
	}
	return sdk.NewContext(ms, header, mode, app.Logger).WithAccountCache(accountCache)
}

type state struct {
	ms           sdk.CacheMultiStore
	AccountCache sdk.AccountCache
	Ctx          sdk.Context
}

func (st *state) CacheMultiStore() sdk.CacheMultiStore {
	return st.ms.CacheMultiStore()
}

func (st *state) WriteAccountCache() {
	st.AccountCache.Write()
}

func (app *BaseApp) SetCheckState(header abci.Header) {
	accountCache := auth.NewAccountCache(app.AccountStoreCache)

	ms := app.cms.CacheMultiStore()
	app.CheckState = &state{
		ms:           ms,
		AccountCache: accountCache,
		Ctx:          sdk.NewContext(ms, header, sdk.RunTxModeCheck, app.Logger).WithAccountCache(accountCache),
	}
}

func (app *BaseApp) SetDeliverState(header abci.Header) {
	accountCache := auth.NewAccountCache(app.AccountStoreCache)

	ms := app.cms.CacheMultiStore()
	app.DeliverState = &state{
		ms:           ms,
		AccountCache: accountCache,
		Ctx:          sdk.NewContext(ms, header, sdk.RunTxModeDeliver, app.Logger).WithAccountCache(accountCache),
	}
}

func (app *BaseApp) SetAccountStoreCache(cdc *codec.Codec, accountStore sdk.KVStore, cap int) {
	app.AccountStoreCache = auth.NewAccountStoreCache(cdc, accountStore, cap)
}

//______________________________________________________________________________

// ABCI

// Implements ABCI
func (app *BaseApp) Info(req abci.RequestInfo) abci.ResponseInfo {
	lastCommitID := app.cms.LastCommitID()

	return abci.ResponseInfo{
		Data:             app.name,
		LastBlockHeight:  lastCommitID.Version,
		LastBlockAppHash: lastCommitID.Hash,
	}
}

// Implements ABCI
func (app *BaseApp) SetOption(req abci.RequestSetOption) (res abci.ResponseSetOption) {
	// TODO: Implement
	return
}

// Implements ABCI
// InitChain runs the initialization logic directly on the CommitMultiStore and commits it.
func (app *BaseApp) InitChain(req abci.RequestInitChain) (res abci.ResponseInitChain) {
	// Initialize the deliver state and check state with ChainID and run initChain
	app.SetDeliverState(abci.Header{ChainID: req.ChainId})
	app.SetCheckState(abci.Header{ChainID: req.ChainId})

	if app.initChainer == nil {
		return
	}
	res = app.initChainer(app.DeliverState.Ctx, req)

	// we need to write updates to underlying cache and storage
	app.DeliverState.WriteAccountCache()

	// NOTE: we don't commit, but BeginBlock for block 1
	// starts from this DeliverState
	return
}

// Filter peers by address / port
func (app *BaseApp) FilterPeerByAddrPort(info string) abci.ResponseQuery {
	if app.addrPeerFilter != nil {
		return app.addrPeerFilter(info)
	}
	return abci.ResponseQuery{}
}

// Filter peers by public key
func (app *BaseApp) FilterPeerByPubKey(info string) abci.ResponseQuery {
	if app.pubkeyPeerFilter != nil {
		return app.pubkeyPeerFilter(info)
	}
	return abci.ResponseQuery{}
}

// Splits a string path using the delimter '/'.  i.e. "this/is/funny" becomes []string{"this", "is", "funny"}
func SplitPath(requestPath string) (path []string) {
	path = strings.Split(requestPath, "/")
	// first element is empty string
	if len(path) > 0 && path[0] == "" {
		path = path[1:]
	}
	return path
}

// Implements ABCI.
// Delegates to CommitMultiStore if it implements Queryable
func (app *BaseApp) Query(req abci.RequestQuery) (res abci.ResponseQuery) {
	path := SplitPath(req.Path)
	if len(path) == 0 {
		msg := "no query path provided"
		return sdk.ErrUnknownRequest(msg).QueryResult()
	}
	switch path[0] {
	// "/app" prefix for special application queries
	case "app":
		return handleQueryApp(app, path, req)
	case "store":
		return handleQueryStore(app, path, req)
	case "p2p":
		return handleQueryP2P(app, path, req)
	case "custom":
		return handleQueryCustom(app, path, req)
	}

	msg := "unknown query path"
	return sdk.ErrUnknownRequest(msg).QueryResult()
}

func handleQueryApp(app *BaseApp, path []string, req abci.RequestQuery) (res abci.ResponseQuery) {
	if len(path) >= 2 {
		var result sdk.Result
		switch path[1] {
		case "simulate":
			txBytes := req.Data
			tx, err := app.TxDecoder(txBytes)
			if err != nil {
				result = err.Result()
			} else {
				result = app.Simulate(txBytes, tx)
			}
		case "version":
			return abci.ResponseQuery{
				Code:  uint32(sdk.ABCICodeOK),
				Value: []byte(version.GetVersion()),
			}
		default:
			result = sdk.ErrUnknownRequest(fmt.Sprintf("Unknown query: %s", path)).Result()
		}

		// Encode with json
		value := codec.Cdc.MustMarshalBinaryLengthPrefixed(result)
		return abci.ResponseQuery{
			Code:  uint32(sdk.ABCICodeOK),
			Value: value,
		}
	}
	msg := "Expected second parameter to be either simulate or version, neither was present"
	return sdk.ErrUnknownRequest(msg).QueryResult()
}

func handleQueryStore(app *BaseApp, path []string, req abci.RequestQuery) (res abci.ResponseQuery) {
	// "/store" prefix for store queries
	queryable, ok := app.cms.(sdk.Queryable)
	if !ok {
		msg := "multistore doesn't support queries"
		return sdk.ErrUnknownRequest(msg).QueryResult()
	}
	req.Path = "/" + strings.Join(path[1:], "/")
	return queryable.Query(req)
}

// nolint: unparam
func handleQueryP2P(app *BaseApp, path []string, req abci.RequestQuery) (res abci.ResponseQuery) {
	// "/p2p" prefix for p2p queries
	if len(path) >= 4 {
		if path[1] == "filter" {
			if path[2] == "addr" {
				return app.FilterPeerByAddrPort(path[3])
			}
			if path[2] == "pubkey" {
				// TODO: this should be changed to `id`
				// NOTE: this changed in tendermint and we didn't notice...
				return app.FilterPeerByPubKey(path[3])
			}
		} else {
			msg := "Expected second parameter to be filter"
			return sdk.ErrUnknownRequest(msg).QueryResult()
		}
	}

	msg := "Expected path is p2p filter <addr|pubkey> <parameter>"
	return sdk.ErrUnknownRequest(msg).QueryResult()
}

func handleQueryCustom(app *BaseApp, path []string, req abci.RequestQuery) (res abci.ResponseQuery) {
	// path[0] should be "custom" because "/custom" prefix is required for keeper queries.
	// the queryRouter routes using path[1]. For example, in the path "custom/gov/proposal", queryRouter routes using "gov"
	if len(path) < 2 || path[1] == "" {
		return sdk.ErrUnknownRequest("No route for custom query specified").QueryResult()
	}
	querier := app.queryRouter.Route(path[1])
	if querier == nil {
		return sdk.ErrUnknownRequest("no custom querier found for route " + path[1]).QueryResult()
	}

	ctx := sdk.NewContext(app.cms.CacheMultiStore(), app.CheckState.Ctx.BlockHeader(), sdk.RunTxModeCheck, app.Logger)
	ctx = ctx.WithAccountCache(auth.NewAccountCache(app.AccountStoreCache))

	// Passes the rest of the path as an argument to the querier.
	// For example, in the path "custom/gov/proposal/test", the gov querier gets []string{"proposal", "test"} as the path
	resBytes, err := querier(ctx, path[2:], req)
	if err != nil {
		return abci.ResponseQuery{
			Code: uint32(err.ABCICode()),
			Log:  err.ABCILog(),
		}
	}
	return abci.ResponseQuery{
		Code:  uint32(sdk.ABCICodeOK),
		Value: resBytes,
	}
}

// BeginBlock implements the ABCI application interface.
func (app *BaseApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	if app.cms.TracingEnabled() {
		app.cms.ResetTraceContext()
		app.cms.WithTracingContext(sdk.TraceContext(
			map[string]interface{}{"blockHeight": req.Header.Height},
		))
	}

	sdk.UpgradeMgr.SetHeight(req.Header.Height)

	// Initialize the DeliverTx state. If this is the first block, it should
	// already be initialized in InitChain. Otherwise app.DeliverState will be
	// nil, since it is reset on Commit.
	if app.DeliverState == nil {
		app.SetDeliverState(req.Header)
		app.DeliverState.Ctx = app.DeliverState.Ctx.WithVoteInfos(req.LastCommitInfo.GetVotes()).WithBlockHash(req.Hash)
	} else {
		// In the first block, app.DeliverState.Ctx will already be initialized
		// by InitChain. Context is now updated with Header information.
		app.DeliverState.Ctx = app.DeliverState.Ctx.WithBlockHash(req.Hash).WithBlockHeader(req.Header).WithBlockHeight(req.Header.Height)
	}

	if app.beginBlocker != nil {
		res = app.beginBlocker(app.DeliverState.Ctx, req)
	}

	return
}

//getTxFromCache returns a decoded transaction and true if found in the cache;
//otherwise return nil, false
func (app *BaseApp) GetTxFromCache(txBytes []byte) (sdk.Tx, bool) {
	if i, ok := app.txMsgCache.Get(string(txBytes)); ok {
		tx, o := i.(sdk.Tx)
		return tx, o
	}
	return nil, false
}

func (app *BaseApp) AddTxToCache(txBytes []byte, tx sdk.Tx) (evicted bool) {
	return app.txMsgCache.Add(string(txBytes), tx)
}

func (app *BaseApp) RemoveTxFromCache(txBytes []byte) {
	app.txMsgCache.Remove(string(txBytes))
}

// CheckTx implements ABCI
// CheckTx runs the "basic checks" to see whether or not a transaction can possibly be executed,
// first decoding, then the ante handler (which checks signatures/fees/ValidateBasic),
// then finally the route match to see whether a handler exists. CheckTx does not run the actual
// Msg handler function(s).
func (app *BaseApp) CheckTx(req abci.RequestCheckTx) (res abci.ResponseCheckTx) {
	var result sdk.Result
	var tx sdk.Tx
	txBytes := req.Tx
	// try to get the Tx first from cache, if succeed, it means it is PreChecked.
	tx, ok := app.GetTxFromCache(txBytes)
	if ok {
		txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
		app.Logger.Debug("Handle CheckTx", "Tx", txHash)
		result = app.RunTx(sdk.RunTxModeCheckAfterPre, tx, txHash)
	} else {
		tx, err := app.TxDecoder(txBytes)
		if err != nil {
			result = err.Result()
		} else {
			app.txMsgCache.Add(string(txBytes), tx) // for recheck
			txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
			app.Logger.Debug("Handle CheckTx", "Tx", txHash)
			result = app.RunTx(sdk.RunTxModeCheck, tx, txHash)
		}
	}

	if !result.IsOK() {
		app.txMsgCache.Remove(string(req.Tx)) //not usable by DeliverTx
	}

	return abci.ResponseCheckTx{
		Code:   uint32(result.Code),
		Data:   result.Data,
		Log:    result.Log,
		Events: result.GetEvents(),
	}
}

func (app *BaseApp) preCheck(txBytes []byte, mode sdk.RunTxMode) sdk.Result {
	var res sdk.Result
	if app.preChecker != nil && !app.txMsgCache.Contains(string(txBytes)) {
		var tx, err = app.TxDecoder(txBytes)
		if err != nil {
			res = err.Result()
		} else {
			res = app.preChecker(getState(app, mode).Ctx, txBytes, tx)
			if res.IsOK() {
				app.txMsgCache.Add(string(txBytes), tx)
			}
		}
	}
	return res
}

// PreCheckTx implements extended ABCI for concurrency
// PreCheckTx would perform decoding, signture and other basic verification
func (app *BaseApp) PreCheckTx(req abci.RequestCheckTx) (res abci.ResponseCheckTx) {
	result := app.preCheck(req.Tx, sdk.RunTxModeCheck)
	return abci.ResponseCheckTx{
		Code:   uint32(result.Code),
		Data:   result.Data,
		Log:    result.Log,
		Events: result.GetEvents(),
	}
}

// ReCheckTx implements ABCI
// ReCheckTx runs the "minimun checks", after the inital check,
// to see whether or not a transaction can possibly be executed.
func (app *BaseApp) ReCheckTx(req abci.RequestCheckTx) (res abci.ResponseCheckTx) {
	// Decode the Tx.
	var result sdk.Result
	txBytes := req.Tx
	tx, ok := app.GetTxFromCache(txBytes)
	if ok {
		result = app.ReRunTx(txBytes, tx)
	} else { // not suppose to enter here actually
		var tx, err = app.TxDecoder(txBytes)
		if err != nil {
			result = err.Result()
		} else {
			result = app.ReRunTx(txBytes, tx)
		}
	}

	return abci.ResponseCheckTx{
		Code:   uint32(result.Code),
		Data:   result.Data,
		Log:    result.Log,
		Events: result.GetEvents(),
	}
}

// Implements ABCI
func (app *BaseApp) DeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
	// Decode the Tx.
	var result sdk.Result
	txBytes := req.Tx
	tx, ok := app.GetTxFromCache(txBytes) //from checkTx
	if ok {
		// here means either the tx has passed PreDeliverTx or CheckTx,
		// no need to verify signature
		txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
		app.Logger.Debug("Handle DeliverTx", "Tx", txHash)
		result = app.RunTx(sdk.RunTxModeDeliverAfterPre, tx, txHash)
	} else {
		var tx, err = app.TxDecoder(txBytes)
		if err != nil {
			result = err.Result()
		} else {
			txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
			app.Logger.Debug("Handle DeliverTx", "Tx", txHash)
			result = app.RunTx(sdk.RunTxModeDeliver, tx, txHash)
		}
	}

	// Even though the Result.Code is not OK, there are still effects,
	// namely fee deductions and sequence incrementing.

	// Tell the blockchain engine (i.e. Tendermint).
	return abci.ResponseDeliverTx{
		Code:   uint32(result.Code),
		Data:   result.Data,
		Log:    result.Log,
		Events: result.GetEvents(),
	}
}

// PreDeliverTx implements extended ABCI for concurrency
// PreCheckTx would perform decoding, signture and other basic verification
func (app *BaseApp) PreDeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
	result := app.preCheck(req.Tx, sdk.RunTxModeDeliver)
	return abci.ResponseDeliverTx{
		Code:   uint32(result.Code),
		Data:   result.Data,
		Log:    result.Log,
		Events: result.GetEvents(),
	}
}

// Basic validator for msgs
func validateBasicTxMsgs(msgs []sdk.Msg) sdk.Error {
	if msgs == nil || len(msgs) != 1 {
		// TODO: probably shouldn't be ErrInternal. Maybe new ErrInvalidMessage, or ?
		return sdk.ErrInternal("Tx.GetMsgs() must return exactly one message")
	}

	for _, msg := range msgs {
		if !sdk.IsMsgTypeSupported(msg.Type()) {
			return sdk.ErrMsgNotSupported(fmt.Sprintf("msg type(%s) is not supported before height %d",
				msg.Type(), sdk.UpgradeMgr.GetMsgTypeHeight(msg.Type())))
		}

		// Validate the Msg.
		err := msg.ValidateBasic()
		if err != nil {
			err = err.WithDefaultCodespace(sdk.CodespaceRoot)
			return err
		}
	}

	return nil
}

// retrieve the context with cache and store the tx bytes and tx hash
func (app *BaseApp) getContextWithCache(mode sdk.RunTxMode, tx sdk.Tx, txHash string) (sdk.Context,
	sdk.CacheMultiStore, sdk.AccountCache) {
	// Get the context
	ctx := getState(app, mode).Ctx.WithTx(tx)
	// Simulate a DeliverTx
	if mode == sdk.RunTxModeSimulate {
		ctx = ctx.WithRunTxMode(mode)
	}

	ms := ctx.MultiStore()
	msCache := ms.CacheMultiStore()
	if msCache.TracingEnabled() {
		msCache = msCache.WithTracingContext(sdk.TraceContext(
			map[string]interface{}{"txHash": txHash},
		)).(sdk.CacheMultiStore)
	}
	accountCache := getAccountCache(app, mode).Cache()

	return ctx.WithMultiStore(msCache).WithAccountCache(accountCache), msCache, accountCache
}

// Iterates through msgs and executes them
func (app *BaseApp) runMsgs(ctx sdk.Context, msgs []sdk.Msg, mode sdk.RunTxMode) (result sdk.Result) {
	// accumulate results
	logs := make([]string, 0, len(msgs))
	var data []byte   // NOTE: we just append them all (?!)
	var tags sdk.Tags // also just append them all
	var events sdk.Events
	var code sdk.ABCICodeType
	for msgIdx, msg := range msgs {
		// Match route.
		msgRoute := msg.Route()
		handler := app.router.Route(msgRoute)
		if handler == nil {
			return sdk.ErrUnknownRequest("Unrecognized Msg type: " + msgRoute).Result()
		}

		msgResult := handler(ctx.WithRunTxMode(mode), msg)
		msgResult.Tags = append(msgResult.Tags, sdk.MakeTag("action", []byte(msg.Type())))

		// Append Data and Tags
		data = append(data, msgResult.Data...)
		tags = append(tags, msgResult.Tags...)
		events = append(events, msgResult.Events...)

		// Stop execution and return on first failed message.
		if !msgResult.IsOK() {
			logs = append(logs, msgResult.Log)
			code = msgResult.Code
			break
		}

		// Construct usable logs in multi-message transactions.
		logs = append(logs, "Msg "+strconv.Itoa(msgIdx)+": "+msgResult.Log)
	}
	// A tx must only contain one msg. If the msg execution is success, record it
	if code == sdk.ABCICodeOK {
		routerName := msgs[0].Route()
		ctx.RouterCallRecord()[routerName] = true
	}
	result = sdk.Result{
		Code: code,
		Data: data,
		Log:  strings.Join(logs, "\n"),
		// TODO: FeeAmount/FeeDenom
		Tags:   tags,
		Events: events,
	}

	return result
}

// Returns the applicantion's DeliverState if app is in runTxModeDeliver,
// otherwise it returns the application's checkstate.
func getState(app *BaseApp, mode sdk.RunTxMode) *state {
	if mode == sdk.RunTxModeCheckAfterPre ||
		mode == sdk.RunTxModeCheck ||
		mode == sdk.RunTxModeSimulate ||
		mode == sdk.RunTxModeReCheck {
		return app.CheckState
	}

	return app.DeliverState
}

// Returns AccountCache of CheckState or DeliverState according to the tx mode
func getAccountCache(app *BaseApp, mode sdk.RunTxMode) sdk.AccountCache {
	if mode == sdk.RunTxModeCheckAfterPre ||
		mode == sdk.RunTxModeCheck ||
		mode == sdk.RunTxModeSimulate ||
		mode == sdk.RunTxModeReCheck {
		return app.CheckState.AccountCache
	}

	return app.DeliverState.AccountCache
}

// RunTx processes a transaction. The transactions is proccessed via an
// anteHandler. txBytes may be nil in some cases, eg. in tests. Also, in the
// future we may support "internal" transactions.
func (app *BaseApp) RunTx(mode sdk.RunTxMode, tx sdk.Tx, txHash string) (result sdk.Result) {
	// meter so we initialize upfront.
	ctx, msCache, accountCache := app.getContextWithCache(mode, tx, txHash)

	defer func() {
		if r := recover(); r != nil {
			log := fmt.Sprintf("recovered: %v\nstack:\n%v", r, string(debug.Stack()))
			result = sdk.ErrInternal(log).Result()
		}

	}()

	var msgs = tx.GetMsgs()
	if err := validateBasicTxMsgs(msgs); err != nil {
		return err.Result()
	}

	// run the ante handler
	ctx = ctx.WithValue(TxHashKey, txHash)
	if app.anteHandler != nil {
		newCtx, result, abort := app.anteHandler(ctx, tx, mode)
		if !newCtx.IsZero() {
			ctx = newCtx
		}

		if abort {
			return result
		}
	}

	var txSrc int64
	if stdTx, ok := tx.(auth.StdTx); ok {
		txSrc = stdTx.GetSource()
	}
	result = app.runMsgs(
		ctx.WithValue(TxSourceKey, txSrc),
		msgs,
		mode)

	if mode == sdk.RunTxModeSimulate {
		return
	}

	// only update state if all messages pass
	if result.IsOK() {
		if mode == sdk.RunTxModeDeliver || mode == sdk.RunTxModeDeliverAfterPre {
			if app.collect.CollectAccountBalance {
				app.Pool.AddAddrs(msgs[0].GetInvolvedAddresses())
			}
			if app.collect.CollectTxs {
				// Should we add all msg here with no distinction ？
				app.Pool.AddTx(tx, txHash)
			}
		}
		accountCache.Write()
		msCache.Write()
	}

	return
}

// RunTx processes a transaction. The transactions is proccessed via an
// anteHandler. txBytes may be nil in some cases, eg. in tests. Also, in the
// future we may support "internal" transactions.
func (app *BaseApp) ReRunTx(txBytes []byte, tx sdk.Tx) (result sdk.Result) {
	// meter so we initialize upfront.
	mode := sdk.RunTxModeReCheck
	txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
	ctx, msCache, accountCache := app.getContextWithCache(mode, tx, txHash)

	defer func() {
		if r := recover(); r != nil {
			log := fmt.Sprintf("recovered: %v\nstack:\n%v", r, string(debug.Stack()))
			result = sdk.ErrInternal(log).Result()
		}

	}()

	// run the ante handler
	if app.anteHandler != nil {
		newCtx, result, abort := app.anteHandler(ctx.WithValue(TxHashKey, txHash), tx, mode)
		if !newCtx.IsZero() {
			ctx = newCtx
		}

		if abort {
			return result
		}
	}

	var msgs = tx.GetMsgs()
	var txSrc int64
	if stdTx, ok := tx.(auth.StdTx); ok {
		txSrc = stdTx.GetSource()
	}
	result = app.runMsgs(ctx.WithValue(TxHashKey, txHash).WithValue(TxSourceKey, txSrc), msgs, mode)

	// only update state if all messages pass
	if result.IsOK() {
		accountCache.Write()
		msCache.Write()
	}

	return
}

// EndBlock implements the ABCI application interface.
func (app *BaseApp) EndBlock(req abci.RequestEndBlock) (res abci.ResponseEndBlock) {
	if app.DeliverState.ms.TracingEnabled() {
		app.DeliverState.ms = app.DeliverState.ms.ResetTraceContext().(sdk.CacheMultiStore)
	}

	if app.endBlocker != nil {
		res = app.endBlocker(app.DeliverState.Ctx, req)
	}

	return
}

// Implements ABCI
func (app *BaseApp) Commit() (res abci.ResponseCommit) {
	header := app.DeliverState.Ctx.BlockHeader()
	/*
		// Write the latest Header to the store
			headerBytes, err := proto.Marshal(&header)
			if err != nil {
				panic(err)
			}
			app.db.SetSync(dbHeaderKey, headerBytes)
	*/

	// Write the Deliver state and commit the MultiStore
	app.DeliverState.WriteAccountCache()
	app.DeliverState.ms.Write()
	commitID := app.cms.Commit()
	// TODO: this is missing a module identifier and dumps byte array
	app.Logger.Debug("Commit synced",
		"commit", commitID,
	)

	// Reset the Check state to the latest committed
	// NOTE: safe because Tendermint holds a lock on the mempool for Commit.
	// Use the header from this latest block.
	app.SetCheckState(header)

	// Empty the Deliver state
	app.DeliverState = nil
	app.Pool.Clear()

	return abci.ResponseCommit{
		Data: commitID.Hash,
	}
}

func (app *BaseApp) StartRecovery(manifest *abci.Manifest) error {
	return app.StateSyncHelper.StartRecovery(manifest)
}

func (app *BaseApp) WriteRecoveryChunk(hash abci.SHA256Sum, chunk *abci.AppStateChunk, isComplete bool) error {
	if err := app.StateSyncHelper.WriteRecoveryChunk(hash, chunk, isComplete); err != nil {
		return err
	}

	if isComplete {
		// load into memory from db
		if err := app.LoadCMSLatestVersion(); err != nil {
			return err
		}
		stores := app.GetCommitMultiStore()
		commitId := stores.LastCommitID()
		hashHex := fmt.Sprintf("%X", commitId.Hash)
		app.Logger.Info("commit by state reactor", "version", commitId.Version, "hash", hashHex)

		// simulate we just "Commit()" :P
		app.SetCheckState(abci.Header{Height: snapshot.Manager().RestorationManifest.Height})
		app.DeliverState = nil
	}
	return nil
}

func (app *BaseApp) GetDB() dbm.DB {
	return app.db
}

func (app *BaseApp) SetPruning(strategy sdk.PruningStrategy) {
	app.cms.SetPruning(strategy)
}
