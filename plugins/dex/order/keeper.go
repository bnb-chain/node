package order

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/wire"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	bc "github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	tmtypes "github.com/tendermint/tendermint/types"

	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	dexTypes "github.com/BiJie/BinanceChain/plugins/dex/types"
)

// in the future, this may be distributed via Sharding
type Keeper struct {
	PairMapper store.TradingPairMapper

	ck bank.Keeper

	storeKey       sdk.StoreKey // The key used to access the store from the Context.
	codespace      sdk.CodespaceType
	engines        map[string]*me.MatchEng
	allOrders      map[string]NewOrderMsg
	roundOrders    map[string]int // limit to the total tx number in a block
	roundIOCOrders map[string][]string
	poolSize       uint // number of concurrent channels, counted in the pow of 2
	cdc            *wire.Codec
}

// Transfer represents a transfer between trade currencies
type Transfer struct {
	account sdk.AccAddress
	inCcy   string
	in      int64
	outCcy  string
	out     int64
	unlock  int64
}

type OrderBookSnapshot struct {
	Buys           []me.PriceLevel `json:"buys"`
	Sells          []me.PriceLevel `json:"sells"`
	LastTradePrice int64           `json:"lasttradeprice"`
}

type ActiveOrders struct {
	Orders []NewOrderMsg `json:"orders"`
}

func CreateMatchEng(lotSize int64) *me.MatchEng {
	return me.NewMatchEng(1000, lotSize, 0.05)
}

func genOrderBookSnapshotKey(height int64, pair string) string {
	return fmt.Sprintf("orderbook_%v_%v", height, pair)
}

func genActiveOrdersSnapshotKey(height int64) string {
	return fmt.Sprintf("activeorders_%v", height)
}

// NewKeeper - Returns the Keeper
func NewKeeper(key sdk.StoreKey, bankKeeper bank.Keeper, tradingPairMapper store.TradingPairMapper, codespace sdk.CodespaceType,
	concurrency uint, cdc *wire.Codec) (*Keeper, error) {
	engines := make(map[string]*me.MatchEng)
	return &Keeper{
		PairMapper:     tradingPairMapper,
		ck:             bankKeeper,
		storeKey:       key,
		codespace:      codespace,
		engines:        engines,
		allOrders:      make(map[string]NewOrderMsg, 1000000),
		roundOrders:    make(map[string]int, 256),
		roundIOCOrders: make(map[string][]string, 256),
		poolSize:       concurrency,
		cdc:            cdc,
	}, nil
}

func (kp *Keeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	eng := CreateMatchEng(pair.LotSize)
	kp.engines[pair.GetSymbol()] = eng
	return eng
}

func (kp *Keeper) AddOrder(msg NewOrderMsg, height int64) (err error) {
	//try update order book first
	symbol := msg.Symbol
	eng, ok := kp.engines[symbol]
	if !ok {
		panic(fmt.Sprintf("match engine of symbol %s doesn't exist", symbol))
	}

	_, err = eng.Book.InsertOrder(msg.Id, msg.Side, height, msg.Price, msg.Quantity)
	if err != nil {
		return err
	}

	kp.allOrders[msg.Id] = msg
	kp.roundOrders[symbol] += 1
	if msg.TimeInForce == TimeInForce.IOC {
		kp.roundIOCOrders[symbol] = append(kp.roundIOCOrders[symbol], msg.Id)
	}
	return nil
}

func (kp *Keeper) RemoveOrder(id string, symbol string, side int8, price int64) (ord me.OrderPart, err error) {
	_, ok := kp.allOrders[id]
	if !ok {
		return me.OrderPart{}, errors.New(fmt.Sprintf("Failed to find order [%v] on symbol [%v]", id, symbol))
	}
	eng, ok := kp.engines[symbol]
	if !ok {
		return me.OrderPart{}, errors.New(fmt.Sprintf("Failed to find order [%v] on symbol [%v]", id, symbol))
	}
	delete(kp.allOrders, id)
	return eng.Book.RemoveOrder(id, side, price)
}

func (kp *Keeper) GetOrder(id string, symbol string, side int8, price int64) (ord me.OrderPart, err error) {
	_, ok := kp.allOrders[id]
	if !ok {
		return me.OrderPart{}, errors.New(fmt.Sprintf("Failed to find order [%v] on symbol [%v]", id, symbol))
	}
	eng, ok := kp.engines[symbol]
	if !ok {
		return me.OrderPart{}, errors.New(fmt.Sprintf("Failed to find order [%v] on symbol [%v]", id, symbol))
	}
	return eng.Book.GetOrder(id, side, price)
}

func (kp *Keeper) OrderExists(id string) (NewOrderMsg, bool) {
	ord, ok := kp.allOrders[id]
	return ord, ok
}

func (kp *Keeper) tradeToTransfers(trade me.Trade, tradeCcy, quoteCcy string) (Transfer, Transfer) {
	seller := kp.allOrders[trade.SId].Sender
	buyer := kp.allOrders[trade.BId].Sender
	// TODO: where is 10^8 stored?
	quoteQty := utils.CalBigNotional(trade.LastPx, trade.LastQty)
	unlock := utils.CalBigNotional(trade.OrigBuyPx, trade.BuyCumQty) - utils.CalBigNotional(trade.OrigBuyPx, trade.BuyCumQty-trade.LastQty)
	return Transfer{seller, quoteCcy, quoteQty, tradeCcy, trade.LastQty, trade.LastQty},
		Transfer{buyer, tradeCcy, trade.LastQty, quoteCcy, quoteQty, unlock}
}

//TODO: should get an even hash
func channelHash(account sdk.AccAddress, bucketNumber int) int {
	return int(account[0]+account[1]) % bucketNumber
}

func (kp *Keeper) matchAndDistributeTrades(wg *sync.WaitGroup, distributeTrade bool) []chan Transfer {
	size := len(kp.roundOrders)
	//size is the number of pairs that have new orders, i.e. it should call match()
	if size == 0 {
		return nil
	}
	channelSize := size >> kp.poolSize
	concurrency := 1 << kp.poolSize
	if size%concurrency != 0 {
		channelSize += 1
	}
	outs := make([][]string, concurrency)
	for i, _ := range outs {
		outs[i] = make([]string, channelSize)
	}
	i, j, t, ii := 0, 0, channelSize, 0
	for k, _ := range kp.roundOrders {
		if i >= t {
			j++
			ii = 0
			t += channelSize
		}
		outs[j][ii] = k
		i++
		ii++
	}
	tradeOuts := make([]chan Transfer, concurrency)
	if distributeTrade {
		for i, _ := range tradeOuts {
			//TODO: channelSize is enough for buffer to facilitate ?
			tradeOuts[i] = make(chan Transfer, channelSize)
		}
	}
	wg.Add(concurrency)
	for i = 0; i < concurrency; i++ {
		channel := outs[i]
		go func() {
			for _, ts := range channel {
				if ts == "" {
					break
				}
				engine := kp.engines[ts]
				if engine.Match() {
					if distributeTrade {
						tradeCcy, quoteCcy, _ := utils.TradeSymbol2Ccy(ts)
						for _, t := range engine.Trades {
							t1, t2 := kp.tradeToTransfers(t, tradeCcy, quoteCcy)
							//TODO: calculate fees as transfer, f1, f2, and push into the tradeOuts
							c := channelHash(t1.account, concurrency)
							tradeOuts[c] <- t1
							c = channelHash(t2.account, concurrency)
							tradeOuts[c] <- t2
						}
					}
					engine.DropFilledOrder()
				} // TODO: when Match() failed, have to unsolicited cancel all the orders
				// when multiple unsolicited cancel happened, the validator would stop running
				// and ask for help
				iocIDs := kp.roundIOCOrders[ts]
				for _, id := range iocIDs {
					if msg, ok := kp.allOrders[id]; ok {
						if ord, err := kp.RemoveOrder(msg.Id, msg.Symbol, msg.Side, msg.Price); err == nil {
							if !distributeTrade {
								continue
							}
							//here is a trick to use the same currency as in and out ccy to simulate cancel
							qty := ord.LeavesQty()
							c := channelHash(msg.Sender, concurrency)
							tradeCcy, _, _ := utils.TradeSymbol2Ccy(msg.Symbol)
							var unlock int64
							if msg.Side == Side.BUY {
								unlock = utils.CalBigNotional(msg.Price, msg.Quantity) - utils.CalBigNotional(msg.Price, msg.Quantity-qty)
							} else {
								unlock = qty
							}
							tradeOuts[c] <- Transfer{msg.Sender, tradeCcy, qty, tradeCcy, qty, unlock}
						}
					}
				}
			}
			wg.Done()
		}()
	}

	return tradeOuts
}

func (kp *Keeper) GetOrderBook(pair string, maxLevels int) []store.OrderBookLevel {
	orderbook := make([]store.OrderBookLevel, maxLevels)

	i, j := 0, 0

	if eng, ok := kp.engines[pair]; ok {
		eng.Book.ShowDepth(maxLevels, func(p *me.PriceLevel) {
			orderbook[i].BuyPrice = utils.Fixed8(p.Price)
			orderbook[i].BuyQty = utils.Fixed8(p.TotalLeavesQty())
			i++
		},
			func(p *me.PriceLevel) {
				orderbook[j].SellPrice = utils.Fixed8(p.Price)
				orderbook[j].SellQty = utils.Fixed8(p.TotalLeavesQty())
				j++
			})
	}
	return orderbook
}

func (kp *Keeper) GetLastTrades(pair string) ([]me.Trade, int64) {
	if eng, ok := kp.engines[pair]; ok {
		return eng.Trades, eng.LastTradePrice
	}
	return nil, 0
}

func (kp *Keeper) ClearOrderBook(pair string) {
	if eng, ok := kp.engines[pair]; ok {
		eng.Book.Clear()
	}
}

func (kp *Keeper) doTransfer(ctx sdk.Context, accountMapper auth.AccountMapper, tran Transfer) sdk.Error {
	//for Out, only need to reduce the locked.
	account := accountMapper.GetAccount(ctx, tran.account).(types.NamedAccount)
	account.SetLockedCoins(
		account.GetLockedCoins().Minus(sdk.Coins{sdk.Coin{Denom: tran.outCcy, Amount: sdk.NewInt(tran.unlock)}}))
	accountMapper.SetAccount(ctx, account)
	//TODO: error handling
	_, _, sdkErr := kp.ck.AddCoins(ctx, tran.account, sdk.Coins{sdk.Coin{Denom: tran.inCcy, Amount: sdk.NewInt(tran.in)}})
	_, _, sdkErr = kp.ck.AddCoins(ctx, tran.account, sdk.Coins{sdk.Coin{Denom: tran.outCcy, Amount: sdk.NewInt(tran.unlock - tran.out)}})
	return sdkErr
}

func (kp *Keeper) clearAfterMatch() (err error) {
	kp.roundOrders = make(map[string]int, 256)
	kp.roundIOCOrders = make(map[string][]string, 256)
	return nil
}

// MatchAll will only concurrently match but do not allocate into accounts
func (kp *Keeper) MatchAll() (code sdk.CodeType, err error) {
	var wgOrd sync.WaitGroup
	tradeOuts := kp.matchAndDistributeTrades(&wgOrd, false) //only match
	if tradeOuts == nil {
		//TODO: logging
		return sdk.CodeOK, nil
	}
	wgOrd.Wait()
	return sdk.CodeOK, nil
}

// MatchAndAllocateAll() is concurrently matching and allocating across
// all the symbols' order books, among all the clients
func (kp *Keeper) MatchAndAllocateAll(ctx sdk.Context, accountMapper auth.AccountMapper) (code sdk.CodeType, err error) {
	var wg sync.WaitGroup
	allocate := func(ctx sdk.Context, accountMapper auth.AccountMapper, c <-chan Transfer) {
		for n := range c {
			kp.doTransfer(ctx, accountMapper, n)
		}
		wg.Done()
	}
	var wgOrd sync.WaitGroup
	tradeOuts := kp.matchAndDistributeTrades(&wgOrd, true)
	if tradeOuts == nil {
		//TODO: logging
		return sdk.CodeOK, nil
	}

	wg.Add(len(tradeOuts))
	for _, c := range tradeOuts {
		go allocate(ctx, accountMapper, c)
	}
	wgOrd.Wait()
	for _, t := range tradeOuts {
		close(t)
	}
	wg.Wait()
	return sdk.CodeOK, nil
}

func (kp *Keeper) ExpireOrders(ctx sdk.Context, height int64, accountMapper auth.AccountMapper) (code sdk.CodeType, err error) {
	return sdk.CodeOK, nil
}

func (kp *Keeper) MarkBreatheBlock(ctx sdk.Context, height, blockTime int64) {
	key := utils.Int642Bytes(blockTime / 1000)
	store := ctx.KVStore(kp.storeKey)
	bz, err := kp.cdc.MarshalBinaryBare(height)
	if err != nil {
		panic(err)
	}
	store.Set([]byte(key), bz)
}

func (kp *Keeper) GetBreatheBlockHeight(timeNow time.Time, kvStore sdk.KVStore, daysBack int) int64 {
	bz := []byte(nil)
	for i := 0; bz == nil && i <= daysBack; i++ {
		t := timeNow.AddDate(0, 0, -i).Unix()
		key := utils.Int642Bytes(t)
		bz = kvStore.Get([]byte(key))
	}
	if bz == nil {
		//TODO: logging
		return 0
	}
	var height int64
	err := kp.cdc.UnmarshalBinaryBare(bz, &height)
	if err != nil {
		panic(err)
	}
	return height
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

func (kp *Keeper) SnapShotOrderBook(ctx sdk.Context, height int64) (err error) {
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

func (kp *Keeper) LoadOrderBookSnapshot(ctx sdk.Context, daysBack int) (int64, error) {
	kvStore := ctx.KVStore(kp.storeKey)
	timeNow := time.Now()
	height := kp.GetBreatheBlockHeight(timeNow, kvStore, daysBack)
	if height == 0 {
		//TODO: Log. this might be the first day online and no breathe block is saved.
		return height, nil
	}

	allPairs := kp.PairMapper.ListAllTradingPairs(ctx)
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

func (kp *Keeper) InitOrderBook(ctx sdk.Context, daysBack int, blockDB dbm.DB, lastHeight int64, txDecoder sdk.TxDecoder) {
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

// Key to knowing the trend on the streets!
var makerFeeKey = []byte("MakerFee")
var takerFeeKey = []byte("TakerFee")
var feeFactorKey = []byte("FeeFactor")
var maxFeeKey = []byte("MaxFee")
var nativeTokenDiscountKey = []byte("NativeTokenDiscount")
var volumeBucketDurationKey = []byte("VolumeBucketDuration")

func itob(num int64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(buf, num)
	b := buf[:n]
	return b
}

func btoi(bytes []byte) int64 {
	x, _ := binary.Varint(bytes)
	return x
}

// GetFees - returns the current fees settings
func (k Keeper) GetFees(ctx sdk.Context) (
	makerFee int64, takerFee int64, feeFactor int64, maxFee int64, nativeTokenDiscount int64, volumeBucketDuration int64,
) {
	store := ctx.KVStore(k.storeKey)
	makerFee = btoi(store.Get(makerFeeKey))
	takerFee = btoi(store.Get(takerFeeKey))
	feeFactor = btoi(store.Get(feeFactorKey))
	maxFee = btoi(store.Get(maxFeeKey))
	nativeTokenDiscount = btoi(store.Get(nativeTokenDiscountKey))
	volumeBucketDuration = btoi(store.Get(volumeBucketDurationKey))
	return makerFee, takerFee, feeFactor, maxFee, nativeTokenDiscount, volumeBucketDuration
}

func (k Keeper) setMakerFee(ctx sdk.Context, makerFee int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(makerFee)
	store.Set(makerFeeKey, b)
}

func (k Keeper) setTakerFee(ctx sdk.Context, takerFee int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(takerFee)
	store.Set(takerFeeKey, b)
}

func (k Keeper) setFeeFactor(ctx sdk.Context, feeFactor int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(feeFactor)
	store.Set(feeFactorKey, b)
}

func (k Keeper) setMaxFee(ctx sdk.Context, maxFee int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(maxFee)
	store.Set(maxFeeKey, b)
}

func (k Keeper) setNativeTokenDiscount(ctx sdk.Context, nativeTokenDiscount int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(nativeTokenDiscount)
	store.Set(nativeTokenDiscountKey, b)
}

func (k Keeper) setVolumeBucketDuration(ctx sdk.Context, volumeBucketDuration int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(volumeBucketDuration)
	store.Set(volumeBucketDurationKey, b)
}

// InitGenesis - store the genesis trend
func (k Keeper) InitGenesis(ctx sdk.Context, data TradingGenesis) {
	k.setMakerFee(ctx, data.MakerFee)
	k.setTakerFee(ctx, data.TakerFee)
	k.setFeeFactor(ctx, data.FeeFactor)
	k.setMaxFee(ctx, data.MaxFee)
	k.setNativeTokenDiscount(ctx, data.NativeTokenDiscount)
	k.setVolumeBucketDuration(ctx, data.VolumeBucketDuration)
}
