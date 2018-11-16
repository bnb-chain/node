package app

import (
	"fmt"
	"io"
	"sync"

	"github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
)

var (
	DefaultCacheSize  = 20000
	DefaultMaxSurvive = 10
)

type SentryApplication struct {
	abci.BaseApplication

	logger log.Logger
	cache  mapTxCache
}

type SentryConfig struct {
	CacheSize  int
	MaxSurvive int
}

var SentryAppConfig = SentryConfig{DefaultCacheSize, DefaultMaxSurvive}

func NewSentryApplication(logger log.Logger, _ db.DB, _ io.Writer) abci.Application {

	return &SentryApplication{
		logger: logger,
		cache:  newMapTxCache(SentryAppConfig.CacheSize),
	}
}

func (app *SentryApplication) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	app.cache.nextRound()
	return abci.ResponseBeginBlock{}
}

func (app *SentryApplication) Info(req abci.RequestInfo) (resInfo abci.ResponseInfo) {
	return abci.ResponseInfo{Data: "{\"name\": \"dumyApp\"}"}
}

func (app *SentryApplication) CheckTx(tx []byte) abci.ResponseCheckTx {
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

func (app *SentryApplication) ReCheckTx(txBytes []byte) (res abci.ResponseCheckTx) {
	// Decode the Tx.
	txHash := common.HexBytes(tmhash.Sum(txBytes)).String()

	if tx := app.cache.get(txHash); tx != nil {
		if tx.survive >= SentryAppConfig.MaxSurvive {
			app.cache.delete(txHash)
			app.logger.Info("Remove tx", "txHash", txHash)
			return abci.ResponseCheckTx{
				Code: uint32(types.CodeInternal),
				Log:  fmt.Sprintf("Tx expires. Hash: %s.", txHash),
			}
		} else {
			// Mark as good
			tx.rechecked = true
		}
	} else {
		if app.cache.size() > SentryAppConfig.CacheSize {
			app.logger.Debug("Remove tx", "txHash", txHash)
			return abci.ResponseCheckTx{
				Code: uint32(types.CodeInternal),
				Log:  fmt.Sprintf("Cache is full, discard tx. Hash: %s", txHash),
			}
		}
		app.logger.Info("Add tx", "txHash", txHash)
		app.cache.add(txHash)
	}

	return abci.ResponseCheckTx{
		Code: abci.CodeTypeOK,
	}
}

// --------------------------------------------------------------------------------
// A mirror of mempool.txs
type mapTxCache struct {
	mtx  sync.Mutex
	pool map[string]*surviveTx
}

type surviveTx struct {
	survive   int
	rechecked bool
}

func newMapTxCache(size int) mapTxCache {
	return mapTxCache{
		pool: make(map[string]*surviveTx, size),
	}
}

func (c mapTxCache) size() int {
	return len(c.pool)
}

func (c mapTxCache) nextRound() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	victims := make([]string, 0)
	for k, tx := range c.pool {
		// Do not rechecked last round, which means marked as bad tx by mempool or already in block.
		if !tx.rechecked {
			victims = append(victims, k)
		} else {
			tx.survive = tx.survive + 1
			// Reset rechecked
			tx.rechecked = false
		}
	}
	for _, v := range victims {
		delete(c.pool, v)
	}
}

func (c mapTxCache) add(hash string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.pool[hash] = &surviveTx{survive: 1, rechecked: true}
}

func (c mapTxCache) delete(hash string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	delete(c.pool, hash)
}

func (c mapTxCache) get(hash string) *surviveTx {
	return c.pool[hash]
}
