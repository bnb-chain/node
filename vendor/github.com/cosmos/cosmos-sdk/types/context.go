package types

import (
	"context"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

type Context struct {
	ctx                context.Context
	ms                 MultiStore
	blockHeader        abci.Header
	blockHeight        int64
	blockHash          []byte
	consParams         *abci.ConsensusParams
	chainID            string
	tx                 Tx
	logger             log.Logger
	voteInfos          []abci.VoteInfo
	mode               RunTxMode
	accountCache       AccountCache
	routerCallRecord   map[string]bool
	eventManager       *EventManager
	sideChainKeyPrefix []byte
	sideChainId        string
}

// create a new context
func NewContext(ms MultiStore, header abci.Header, runTxMode RunTxMode, logger log.Logger) Context {
	return Context{
		ctx:              context.Background(),
		ms:               ms,
		blockHeader:      header,
		blockHeight:      header.Height,
		chainID:          header.ChainID,
		mode:             runTxMode,
		logger:           logger,
		routerCallRecord: make(map[string]bool),
		eventManager:     NewEventManager(),
	}
}

func (c Context) Context() context.Context {
	return c.ctx
}

func (c Context) MultiStore() MultiStore {
	return c.ms
}

func (c Context) BlockHeader() abci.Header {
	return c.blockHeader
}

func (c Context) BlockHeight() int64 {
	return c.blockHeight
}

func (c Context) BlockHash() []byte {
	return c.blockHash
}

func (c Context) ConsensusParams() *abci.ConsensusParams {
	return c.consParams
}

func (c Context) ChainID() string {
	return c.chainID
}

func (c Context) Tx() Tx {
	return c.tx
}

func (c Context) Logger() log.Logger {
	return c.logger
}

func (c Context) VoteInfos() []abci.VoteInfo {
	return c.voteInfos
}

func (c Context) IsCheckTx() bool {
	return c.mode == RunTxModeCheck || c.mode == RunTxModeCheckAfterPre
}

func (c Context) IsReCheckTx() bool {
	return c.mode == RunTxModeReCheck
}

func (c Context) IsDeliverTx() bool {
	return c.mode == RunTxModeDeliver || c.mode == RunTxModeDeliverAfterPre
}

func (c Context) AccountCache() AccountCache {
	return c.accountCache
}

func (c Context) RouterCallRecord() map[string]bool {
	return c.routerCallRecord
}

func (c Context) EventManager() *EventManager {
	return c.eventManager
}

func (c Context) SideChainId() string {
	return c.sideChainId
}

//----------------------------------------
// With* (setting a value)

func (c Context) WithContext(ctx context.Context) Context {
	c.ctx = ctx
	return c
}

func (c Context) WithMultiStore(ms MultiStore) Context {
	c.ms = ms
	return c
}

func (c Context) WithBlockHash(hash []byte) Context {
	c.blockHash = hash
	return c
}

func (c Context) WithBlockHeader(header abci.Header) Context {
	c.blockHeader = header
	return c
}

func (c Context) WithBlockTime(newTime time.Time) Context {
	newHeader := c.BlockHeader()
	newHeader.Time = newTime
	return c.WithBlockHeader(newHeader)
}

func (c Context) WithProposer(addr ConsAddress) Context {
	newHeader := c.BlockHeader()
	newHeader.ProposerAddress = addr.Bytes()
	return c.WithBlockHeader(newHeader)
}

func (c Context) WithBlockHeight(height int64) Context {
	newHeader := c.BlockHeader()
	newHeader.Height = height
	c.blockHeight = height
	return c.WithBlockHeader(newHeader)
}

func (c Context) WithConsensusParams(params *abci.ConsensusParams) Context {
	if params == nil {
		return c
	}
	c.consParams = params
	return c
}

func (c Context) WithChainID(chainID string) Context {
	c.chainID = chainID
	return c
}

func (c Context) WithTx(tx Tx) Context {
	c.tx = tx
	return c
}

func (c Context) WithLogger(logger log.Logger) Context {
	c.logger = logger
	return c
}

func (c Context) WithVoteInfos(voteInfos []abci.VoteInfo) Context {
	c.voteInfos = voteInfos
	return c
}

func (c Context) WithRunTxMode(runTxMode RunTxMode) Context {
	c.mode = runTxMode
	return c
}

func (c Context) WithAccountCache(cache AccountCache) Context {
	c.accountCache = cache
	return c
}

func (c Context) WithRouterCallRecord(record map[string]bool) Context {
	c.routerCallRecord = record
	return c
}

func (c Context) DepriveSideChainKeyPrefix() Context {
	c.sideChainKeyPrefix = nil
	c.sideChainId = ""
	return c
}

func (c Context) WithEventManager(em *EventManager) Context {
	c.eventManager = em
	return c
}

func (c Context) WithSideChainKeyPrefix(prefix []byte) Context {
	c.sideChainKeyPrefix = prefix
	return c
}

func (c Context) WithSideChainId(sideChainId string) Context {
	c.sideChainId = sideChainId
	return c
}

// is context nil
func (c Context) IsZero() bool {
	return c.ctx == nil && c.ms == nil
}

// WithValue is deprecated, provided for backwards compatibility
// Please use
//     ctx = ctx.WithContext(context.WithValue(ctx.Context(), key, false))
// instead of
//     ctx = ctx.WithValue(key, false)
func (c Context) WithValue(key, value interface{}) Context {
	c.ctx = context.WithValue(c.ctx, key, value)
	return c
}

// Value is deprecated, provided for backwards compatibility
// Please use
//     ctx.Context().Value(key)
// instead of
//     ctx.Value(key)
func (c Context) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

// ----------------------------------------------------------------------------
// Store / Caching
// ----------------------------------------------------------------------------

// KVStore fetches a KVStore from the MultiStore.
func (c Context) KVStore(key StoreKey) KVStore {
	kvStore := c.MultiStore().GetKVStore(key)
	if c.sideChainKeyPrefix != nil {
		return kvStore.Prefix(c.sideChainKeyPrefix)
	}
	return kvStore
}

// TransientStore fetches a TransientStore from the MultiStore.
func (c Context) TransientStore(key StoreKey) KVStore {
	return c.MultiStore().GetKVStore(key)
}

// Cache the multistore and return a new cached context. The cached context is
// written to the context when writeCache is called.
func (c Context) CacheContext() (cc Context, writeCache func()) {
	cms := c.MultiStore().CacheMultiStore()
	accountCache := c.AccountCache().Cache()

	cc = c.WithMultiStore(cms).WithAccountCache(accountCache)
	return cc, func() {
		accountCache.Write()
		cms.Write()
	}
}
