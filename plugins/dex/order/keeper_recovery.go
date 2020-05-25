package order

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/state"
	tmstore "github.com/tendermint/tendermint/store"
	tmtypes "github.com/tendermint/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/wire"
)

type OrderBookSnapshot struct {
	Buys            []me.PriceLevel `json:"buys"`
	Sells           []me.PriceLevel `json:"sells"`
	LastTradePrice  int64           `json:"lasttradeprice"`
	LastMatchHeight int64           `json:"lastmatchheight"`
}

type ActiveOrders struct {
	Orders []OrderInfo `json:"orders"`
}

func genOrderBookSnapshotKey(height int64, pair string) string {
	return fmt.Sprintf("orderbook_%v_%v", height, pair)
}

func genActiveOrdersSnapshotKey(height int64) string {
	return fmt.Sprintf("activeorders_%v", height)
}

func compressAndSave(snapshot interface{}, cdc *wire.Codec, key string, kv sdk.KVStore) error {
	bytes, err := cdc.MarshalBinaryLengthPrefixed(snapshot)
	if err != nil {
		panic(err)
	}
	compressedBytes, err := utils.Compress(bytes)
	if err != nil {
		panic(err)
	}
	bnclog.Debug(fmt.Sprintf("compressAndSave key: %s, value: %v", key, bytes))
	kv.Set([]byte(key), compressedBytes)
	return nil
}

func (kp *DexKeeper) SnapShotOrderBook(ctx sdk.Context, height int64) (effectedStoreKeys []string, err error) {
	kvstore := ctx.KVStore(kp.storeKey)
	effectedStoreKeys = make([]string, 0)
	for pair, eng := range kp.engines {
		buys, sells := eng.Book.GetAllLevels()
		var snapshot OrderBookSnapshot
		snapshot = OrderBookSnapshot{Buys: buys, Sells: sells, LastTradePrice: eng.LastTradePrice}
		if sdk.IsUpgrade(upgrade.BEP8) {
			snapshot.LastMatchHeight = eng.LastMatchHeight
		}
		key := genOrderBookSnapshotKey(height, pair)
		effectedStoreKeys = append(effectedStoreKeys, key)
		err := compressAndSave(snapshot, kp.cdc, key, kvstore)
		if err != nil {
			return nil, err
		}
		ctx.Logger().Info("Compressed and Saved order book snapshot", "pair", pair)
	}

	msgKeys := make([]string, 0)
	idSymbolMap := make(map[string]string)
	allOrders := kp.GetAllOrders()
	for symbol, orderMap := range allOrders {
		for id := range orderMap {
			idSymbolMap[id] = symbol
			msgKeys = append(msgKeys, id)
		}
	}
	sort.Strings(msgKeys)
	msgs := make([]OrderInfo, len(msgKeys), len(msgKeys))
	for i, key := range msgKeys {
		msgs[i] = *allOrders[idSymbolMap[key]][key]
	}

	snapshot := ActiveOrders{Orders: msgs}
	key := genActiveOrdersSnapshotKey(height)
	effectedStoreKeys = append(effectedStoreKeys, key)
	ctx.Logger().Info("Saving active orders", "height", height)
	return effectedStoreKeys, compressAndSave(snapshot, kp.cdc, key, kvstore)
}

func (kp *DexKeeper) LoadOrderBookSnapshot(ctx sdk.Context, latestBlockHeight int64, timeOfLatestBlock time.Time, blockInterval, daysBack int) (int64, error) {
	height := kp.GetLastBreatheBlockHeight(ctx, latestBlockHeight, timeOfLatestBlock, blockInterval, daysBack)
	ctx.Logger().Info("Loading order book snapshot from last breathe block", "blockHeight", height)
	allPairs := kp.PairMapper.ListAllTradingPairs(ctx)
	if height == 0 {
		// just initialize engines for all pairs
		for _, pair := range allPairs {
			_, ok := kp.engines[pair.GetSymbol()]
			if !ok {
				kp.AddEngine(pair)
			}
		}
		ctx.Logger().Info("No breathe block is ever saved. just created match engines for all the pairs.")
		return height, nil
	}

	upgrade.Mgr.SetHeight(height)
	kvStore := ctx.KVStore(kp.storeKey)
	for _, pair := range allPairs {
		symbol := pair.GetSymbol()
		eng, ok := kp.engines[symbol]
		if !ok {
			eng = kp.AddEngine(pair)
		}

		key := genOrderBookSnapshotKey(height, symbol)
		bz := kvStore.Get([]byte(key))
		if bz == nil {
			// maybe that is a new listed pair
			ctx.Logger().Info("Pair is newly listed, no order book snapshot was saved", "pair", key)
			continue
		}
		b := bytes.NewBuffer(bz)
		var bw bytes.Buffer
		r, err := zlib.NewReader(b)
		if err != nil {
			panic(fmt.Sprintf("failed to unzip snapshort for orderbook [%s]", key))
		}
		io.Copy(&bw, r)
		var ob OrderBookSnapshot
		err = kp.cdc.UnmarshalBinaryLengthPrefixed(bw.Bytes(), &ob)
		if err != nil {
			panic(fmt.Sprintf("failed to unmarshal snapshort for orderbook [%s]", key))
		}
		for _, pl := range ob.Buys {
			eng.Book.InsertPriceLevel(&pl, me.BUYSIDE)
		}
		for _, pl := range ob.Sells {
			eng.Book.InsertPriceLevel(&pl, me.SELLSIDE)
		}
		eng.LastTradePrice = ob.LastTradePrice
		if sdk.IsUpgrade(upgrade.BEP8) {
			eng.LastMatchHeight = ob.LastMatchHeight
		} else {
			eng.LastMatchHeight = height
		}
		ctx.Logger().Info("Successfully Loaded order snapshot", "pair", pair)
	}
	key := genActiveOrdersSnapshotKey(height)
	bz := kvStore.Get([]byte(key))
	if bz == nil {
		ctx.Logger().Info("Pair is newly listed, no active order snapshot was saved", "pair", key)
		return height, nil
	}
	b := bytes.NewBuffer(bz)
	var bw bytes.Buffer
	r, err := zlib.NewReader(b)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal snapshort for active order [%s]", key))
	}
	io.Copy(&bw, r)
	var ao ActiveOrders
	err = kp.cdc.UnmarshalBinaryLengthPrefixed(bw.Bytes(), &ao)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal snapshort for active orders [%s]", key))
	}
	for _, m := range ao.Orders {
		orderHolder := m
		symbol := strings.ToUpper(m.Symbol)
		kp.ReloadOrder(symbol, &orderHolder, height)
	}
	ctx.Logger().Info("Recovered active orders. Snapshot is fully loaded")
	return height, nil
}

func (kp *DexKeeper) replayOneBlocks(logger log.Logger, block *tmtypes.Block, stateDB dbm.DB, txDecoder sdk.TxDecoder,
	height int64, timestamp time.Time) {
	if block == nil {
		logger.Error("No block is loaded. Ignore replay for orderbook")
		return
	}
	abciRes, err := state.LoadABCIResponses(stateDB, height)
	if err != nil {
		panic(fmt.Errorf("failed to load abci response when replay block at height %d, err %v", height, err))
	}
	if abciRes != nil && len(abciRes.DeliverTx) != len(block.Txs) {
		panic(fmt.Errorf("length of delivertx %d and lenght of tx %d mismatch ", len(abciRes.DeliverTx), len(block.Txs)))
	}
	// the time we replay should be consistent with ctx.BlockHeader().Time
	t := timestamp.UnixNano()
	for idx, txBytes := range block.Txs {
		if abciRes.DeliverTx[idx].IsErr() {
			logger.Info("Skip tx when replay", "height", height, "idx", idx)
			continue
		}
		tx, err := txDecoder(txBytes)
		if err != nil {
			panic(err)
		}
		txHash := cmn.HexBytes(tmhash.Sum(txBytes))
		msgs := tx.GetMsgs()
		for _, m := range msgs {
			switch msg := m.(type) {
			case NewOrderMsg:
				var txSource int64
				upgrade.UpgradeBEP10(nil, func() {
					if stdTx, ok := tx.(auth.StdTx); ok {
						txSource = stdTx.GetSource()
					} else {
						logger.Error("tx is not an auth.StdTx", "txhash", txHash.String())
					}
				})
				orderInfo := OrderInfo{
					msg,
					height, t,
					height, t,
					0, txHash.String(), txSource}
				kp.AddOrder(orderInfo, true)
				logger.Info("Added Order", "order", msg)
			case CancelOrderMsg:
				err := kp.RemoveOrder(msg.RefId, msg.Symbol, func(ord me.OrderPart) {
					if kp.CollectOrderInfoForPublish {
						bnclog.Debug("deleted order from order changes map", "orderId", msg.RefId, "isRecovery", true)
						kp.RemoveOrderInfosForPub(msg.Symbol, msg.RefId)
					}
				})
				if err != nil {
					logger.Error("Failed to replay cancel msg", "err", err)
				}
				logger.Info("Canceled Order", "order", msg)
			}
		}
	}
	logger.Info("replayed all tx. Starting match", "height", height)
	kp.MatchSymbols(height, t, false) //no need to check result
}

func (kp *DexKeeper) ReplayOrdersFromBlock(ctx sdk.Context, bc *tmstore.BlockStore, stateDb dbm.DB, lastHeight, breatheHeight int64,
	txDecoder sdk.TxDecoder) error {
	for i := breatheHeight + 1; i <= lastHeight; i++ {
		block := bc.LoadBlock(i)
		ctx.Logger().Info("Relaying block for order book", "height", i)
		upgrade.Mgr.SetHeight(i)
		kp.replayOneBlocks(ctx.Logger(), block, stateDb, txDecoder, i, block.Time)
	}
	return nil
}

func (kp *DexKeeper) initOrderBook(ctx sdk.Context, blockInterval, daysBack int, blockStore *tmstore.BlockStore, stateDB dbm.DB, lastHeight int64, txDecoder sdk.TxDecoder) {
	var timeOfLatestBlock time.Time
	if lastHeight == 0 {
		timeOfLatestBlock = utils.Now()
	} else {
		block := blockStore.LoadBlock(lastHeight)
		timeOfLatestBlock = block.Time
	}
	height, err := kp.LoadOrderBookSnapshot(ctx, lastHeight, timeOfLatestBlock, blockInterval, daysBack)
	if err != nil {
		panic(err)
	}
	logger := ctx.Logger().With("module", "dex")
	logger.Info("Initialized Block Store for replay", "fromHeight", height, "toHeight", lastHeight)
	err = kp.ReplayOrdersFromBlock(ctx.WithLogger(logger), blockStore, stateDB, lastHeight, height, txDecoder)
	if err != nil {
		panic(err)
	}
}
