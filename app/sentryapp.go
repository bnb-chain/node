package app

import (
	"fmt"
	"io"
	"sync"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/stake"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/bnb-chain/node/common/utils"
	"github.com/bnb-chain/node/wire"
)

var (
	DefaultCacheSize  = 20000
	DefaultMaxSurvive = 10

	defaultPower = utils.Fixed8One
)

type SentryApplication struct {
	abci.BaseApplication
	Codec *wire.Codec

	logger log.Logger
	cache  mapTxCache
}

type SentryConfig struct {
	CacheSize  int
	MaxSurvive int
}

var SentryAppConfig = SentryConfig{DefaultCacheSize, DefaultMaxSurvive}

func NewSentryApplication(logger log.Logger, _ db.DB, _ io.Writer) abci.Application {
	var cdc = Codec

	return &SentryApplication{
		Codec:  cdc,
		logger: logger,
		cache:  newMapTxCache(SentryAppConfig.CacheSize),
	}
}

func (app *SentryApplication) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	app.cache.nextRound()
	return abci.ResponseBeginBlock{}
}

func (app *SentryApplication) Info(req abci.RequestInfo) (resInfo abci.ResponseInfo) {
	return abci.ResponseInfo{Data: "{\"name\": \"sentry node\"}"}
}

func (app *SentryApplication) CheckTx(req abci.RequestCheckTx) abci.ResponseCheckTx {
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

func (app *SentryApplication) InitChain(req abci.RequestInitChain) (res abci.ResponseInitChain) {
	stateJSON := req.AppStateBytes

	genesisState := new(GenesisState)
	err := app.Codec.UnmarshalJSON(stateJSON, genesisState)
	if err != nil {
		panic(err)
	}
	validators := make([]abci.ValidatorUpdate, 0)
	if len(genesisState.GenTxs) > 0 {
		for _, genTx := range genesisState.GenTxs {
			var tx auth.StdTx
			err = app.Codec.UnmarshalJSON(genTx, &tx)
			if err != nil {
				panic(err)
			}
			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				switch msg := msg.(type) {
				case stake.MsgCreateValidatorProposal:
					validators = append(validators, abci.ValidatorUpdate{PubKey: tmtypes.TM2PB.PubKey(msg.PubKey), Power: defaultPower.ToInt64()})
				default:
					app.logger.Info("MsgType %s not supported ", msg.Type())
				}
			}
		}
	}
	return abci.ResponseInitChain{
		Validators: validators,
	}
}

func (app *SentryApplication) ReCheckTx(req abci.RequestCheckTx) (res abci.ResponseCheckTx) {
	// Decode the Tx.
	txHash := common.HexBytes(tmhash.Sum(req.Tx)).String()

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
