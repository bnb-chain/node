package order

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bc "github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/BiJie/BinanceChain/common/types"

	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/wire"
)

type OrderBookSnapshot struct {
	Buys           []me.PriceLevel `json:"buys"`
	Sells          []me.PriceLevel `json:"sells"`
	LastTradePrice int64           `json:"lasttradeprice"`
}

type ActiveOrders struct {
	Orders []NewOrderMsg `json:"orders"`
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
	kv.Set([]byte(key), bytes)
	w.Close()
	return nil
}

func (kp *Keeper) SnapShotOrderBook(ctx types.Context, height int64) (err error) {
	kvstore := ctx.KVStore(kp.storeKey)
	for pair, eng := range kp.engines {
		buys, sells := eng.Book.GetAllLevels()
		snapshot := OrderBookSnapshot{Buys: buys, Sells: sells, LastTradePrice: eng.LastTradePrice}
		key := genOrderBookSnapshotKey(height, pair)
		err := compressAndSave(snapshot, kp.cdc, key, kvstore)
		if err != nil {
			return err
		}
	}
	msgs := make([]NewOrderMsg, 0, len(kp.allOrders))
	for _, value := range kp.allOrders {
		msgs = append(msgs, value)
	}
	snapshot := ActiveOrders{Orders: msgs}
	key := genActiveOrdersSnapshotKey(height)
	return compressAndSave(snapshot, kp.cdc, key, kvstore)
}

func (kp *Keeper) LoadOrderBookSnapshot(ctx types.Context, daysBack int) (int64, error) {
	kvStore := ctx.KVStore(kp.storeKey)
	timeNow := time.Now()
	height := kp.GetBreatheBlockHeight(timeNow, kvStore, daysBack)
	allPairs := kp.PairMapper.ListAllTradingPairs(ctx)
	if height == 0 {
		// just initialize engines for all pairs
		for _, pair := range allPairs {
			_, ok := kp.engines[pair.GetSymbol()]
			if !ok {
				kp.AddEngine(pair)
			}
		}
		//TODO: Log. this might be the first day online and no breathe block is saved.
		return height, nil
	}

	for _, pair := range allPairs {
		eng, ok := kp.engines[pair.GetSymbol()]
		if !ok {
			eng = kp.AddEngine(pair)
		}

		key := genOrderBookSnapshotKey(height, pair.GetSymbol())
		bz := kvStore.Get([]byte(key))
		if bz == nil {
			// maybe that is a new listed pair
			//TODO: logging
			continue
		}
		b := bytes.NewBuffer(bz)
		var bw bytes.Buffer
		r, err := zlib.NewReader(b)
		if err != nil {
			continue
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
	}
	key := genActiveOrdersSnapshotKey(height)
	bz := kvStore.Get([]byte(key))
	if bz == nil {
		//TODO: log
		return height, nil
	}
	b := bytes.NewBuffer(bz)
	var bw bytes.Buffer
	r, err := zlib.NewReader(b)
	if err != nil {
		//TODO: log
		return height, nil
	}
	io.Copy(&bw, r)
	var ao ActiveOrders
	err = kp.cdc.UnmarshalBinary(bw.Bytes(), &ao)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal snapshort for active orders [%s]", key))
	}
	for _, m := range ao.Orders {
		kp.allOrders[m.Id] = m
	}
	return height, nil
}

func (kp *Keeper) replayOneBlocks(block *tmtypes.Block, txDecoder sdk.TxDecoder, height int64) {
	if block == nil {
		//TODO: Log
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
				kp.AddOrder(msg, height)
			case CancelOrderMsg:
				ord, ok := kp.allOrders[msg.RefId]
				if !ok {
					panic(fmt.Sprintf("Failed to replay cancel msg on id[%s]", msg.RefId))
				}
				_, err := kp.RemoveOrder(ord.Id, ord.Symbol, ord.Side, ord.Price)
				if err != nil {
					panic(fmt.Sprintf("Failed to replay cancel msg on id[%s]", msg.RefId))
				}
			}
		}
	}
	kp.MatchAll() //no need to check result
}

func (kp *Keeper) ReplayOrdersFromBlock(bc *bc.BlockStore, lastHeight, breatheHeight int64,
	txDecoder sdk.TxDecoder) error {
	for i := breatheHeight + 1; i <= lastHeight; i++ {
		block := bc.LoadBlock(i)
		kp.replayOneBlocks(block, txDecoder, i)
	}
	return nil
}

func (kp *Keeper) InitOrderBook(ctx types.Context, daysBack int, blockDB dbm.DB, lastHeight int64, txDecoder sdk.TxDecoder) {
	height, err := kp.LoadOrderBookSnapshot(ctx, daysBack)
	if err != nil {
		panic(err)
	}

	blockStore := bc.NewBlockStore(blockDB)
	err = kp.ReplayOrdersFromBlock(blockStore, lastHeight, height, txDecoder)
	if err != nil {
		panic(err)
	}
}
