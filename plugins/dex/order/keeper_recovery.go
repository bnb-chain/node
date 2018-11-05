package order

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"sort"
	"time"

	bc "github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	tmtypes "github.com/tendermint/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	bnclog "github.com/BiJie/BinanceChain/common/log"
	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/wire"
)

type OrderBookSnapshot struct {
	Buys           []me.PriceLevel `json:"buys"`
	Sells          []me.PriceLevel `json:"sells"`
	LastTradePrice int64           `json:"lasttradeprice"`
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
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	bytes, err := cdc.MarshalBinary(snapshot)
	if err != nil {
		panic(err)
	}
	_, err = w.Write(bytes)
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	bytes = b.Bytes()
	bnclog.Debug(fmt.Sprintf("compressAndSave key: %s, value: %v", key, bytes))
	kv.Set([]byte(key), bytes)
	w.Close()
	return nil
}

func (kp *Keeper) SnapShotOrderBook(ctx sdk.Context, height int64) (effectedStoreKeys []string, err error) {
	kvstore := ctx.KVStore(kp.storeKey)
	effectedStoreKeys = make([]string, 0)
	for pair, eng := range kp.engines {
		buys, sells := eng.Book.GetAllLevels()
		snapshot := OrderBookSnapshot{Buys: buys, Sells: sells, LastTradePrice: eng.LastTradePrice}
		key := genOrderBookSnapshotKey(height, pair)
		effectedStoreKeys = append(effectedStoreKeys, key)
		err := compressAndSave(snapshot, kp.cdc, key, kvstore)
		if err != nil {
			return nil, err
		}
		logger.Info("Compressed and Saved order book snapshot", "pair", pair)
	}

	msgs := make([]NewOrderMsg, len(kp.allOrders), len(kp.allOrders))
	msgKeys := make([]string, len(kp.allOrders), len(kp.allOrders))
	i := 0
	for key := range kp.allOrders {
		msgKeys[i] = key
		i++
	}
	sort.Strings(msgKeys)
	msgs := make([]OrderInfo, len(msgKeys), len(msgKeys))
	for i, key := range msgKeys {
		msgs[i] = *kp.allOrders[idSymbolMap[key]][key]
	}

	snapshot := ActiveOrders{Orders: msgs}
	key := genActiveOrdersSnapshotKey(height)
	effectedStoreKeys = append(effectedStoreKeys, key)
	logger.Info("Saving active orders", "height", height)
	return effectedStoreKeys, compressAndSave(snapshot, kp.cdc, key, kvstore)
}

func (kp *Keeper) LoadOrderBookSnapshot(ctx sdk.Context, daysBack int) (int64, error) {
	timeNow := utils.Now()
	height := kp.getLastBreatheBlockHeight(ctx, timeNow, daysBack)
	logger.Info("Loading order book snapshot from last breathe block", "blockHeight", height)
	allPairs := kp.PairMapper.ListAllTradingPairs(ctx)
	if height == 0 {
		// just initialize engines for all pairs
		for _, pair := range allPairs {
			_, ok := kp.engines[pair.GetSymbol()]
			if !ok {
				kp.AddEngine(pair)
			}
		}
		logger.Info("No breathe block is ever saved. just created match engines for all the pairs.")
		return height, nil
	}

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
			logger.Info("Pair is newly listed, no order book snapshot was saved", "pair", key)
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
		err = kp.cdc.UnmarshalBinary(bw.Bytes(), &ob)
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
		logger.Info("Successfully Loaded order snapshot", "pair", pair)
	}
	kp.updateLastTradePrices()
	key := genActiveOrdersSnapshotKey(height)
	bz := kvStore.Get([]byte(key))
	if bz == nil {
		logger.Info("Pair is newly listed, no active order snapshot was saved", "pair", key)
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
	err = kp.cdc.UnmarshalBinary(bw.Bytes(), &ao)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal snapshort for active orders [%s]", key))
	}
	for _, m := range ao.Orders {
		orderHolder := m
		kp.allOrders[m.Symbol][m.Id] = &orderHolder
		if kp.CollectOrderInfoForPublish {
			if _, exists := kp.OrderChangesMap[m.Id]; !exists {
				bnclog.Debug("add order to order changes map, during load snapshot, from active orders", "orderId", m.Id)
				kp.OrderChangesMap[m.Id] = &m
			}
		}
	}
	logger.Info("Recovered active orders. Snapshot is fully loaded")
	return height, nil
}

func (kp *Keeper) replayOneBlocks(block *tmtypes.Block, txDecoder sdk.TxDecoder,
	height int64, timestamp time.Time) {
	if block == nil {
		logger.Error("No block is loaded. Ignore replay for orderbook")
		return
	}
	for _, txBytes := range block.Txs {
		tx, err := txDecoder(txBytes)
		if err != nil {
			panic(err)
		}
		msgs := tx.GetMsgs()
		for _, m := range msgs {
			switch msg := m.(type) {
			case NewOrderMsg:
				txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
				// the time we replay should be consistent with ctx.BlockHeader().Time
				// TODO(#118): after upgrade to tendermint 0.24 we should have better and more consistent time representation
				t := timestamp.Unix()
				orderInfo := OrderInfo{
					msg,
					height, t,
					height, t,
					0, txHash}
				kp.AddOrder(orderInfo, true)
				logger.Info("Added Order", "order", msg)
			case CancelOrderMsg:
				err := kp.RemoveOrder(msg.RefId, msg.Symbol, func(ord me.OrderPart) {
					if kp.CollectOrderInfoForPublish {
						bnclog.Debug("deleted order from order changes map", "orderId", msg.Id, "isRecovery", true)
						delete(kp.OrderChangesMap, msg.Id)
					}
				})
				if err != nil {
					panic(fmt.Sprintf("Failed to replay cancel msg for: [%s]", err.Error()))
				}
				logger.Info("Canceled Order", "order", msg)
			}
		}
	}
	logger.Info("replayed all tx. Starting match", "height", height)
	kp.MatchAll(height, timestamp.UnixNano()) //no need to check result
}

func (kp *Keeper) ReplayOrdersFromBlock(bc *bc.BlockStore, lastHeight, breatheHeight int64,
	txDecoder sdk.TxDecoder) error {
	for i := breatheHeight + 1; i <= lastHeight; i++ {
		block := bc.LoadBlock(i)
		logger.Info("Relaying block for order book", "height", i)
		kp.replayOneBlocks(block, txDecoder, i, block.Time)
	}
	return nil
}

func (kp *Keeper) InitOrderBook(ctx sdk.Context, daysBack int, blockDB dbm.DB, lastHeight int64, txDecoder sdk.TxDecoder) {
	defer blockDB.Close()
	height, err := kp.LoadOrderBookSnapshot(ctx, daysBack)
	if err != nil {
		panic(err)
	}
	blockStore := bc.NewBlockStore(blockDB)
	logger.Info("Initialized Block Store for replay")
	err = kp.ReplayOrdersFromBlock(blockStore, lastHeight, height, txDecoder)
	if err != nil {
		panic(err)
	}
}
