package order

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
)

// in the future, this may be distributed via Sharding
type Keeper struct {
	ck             bank.Keeper
	storeKey       sdk.StoreKey // The key used to access the store from the Context.
	codespace      sdk.CodespaceType
	engines        map[string]*me.MatchEng
	allOrders      map[string]NewOrderMsg
	roundOrders    map[string]int // limit to the total tx number in a block
	roundIOCOrders map[string][]string
	poolSize       uint // number of concurrent channels, counted in the pow of 2
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

// NewKeeper - Returns the Keeper
func NewKeeper(key sdk.StoreKey, bankKeeper bank.Keeper, codespace sdk.CodespaceType, concurrency uint) (*Keeper, error) {
	engines := make(map[string]*me.MatchEng)
	allPairs := make([]string, 2)
	for _, p := range allPairs {
		eng := CreateMatchEng(p)
		if err := initializeOrderBook(p, eng); err != nil {
			return nil, err
		}
		engines[p] = eng
	}
	return &Keeper{ck: bankKeeper, storeKey: key, codespace: codespace,
		engines: engines, allOrders: make(map[string]NewOrderMsg, 1000000),
		roundOrders: make(map[string]int, 256), roundIOCOrders: make(map[string][]string, 256), poolSize: concurrency}, nil
}

func CreateMatchEng(symbol string) *me.MatchEng {
	//TODO: read lot size
	return me.NewMatchEng(1000, 1, 0.05)
}

func initializeOrderBook(symbol string, eng *me.MatchEng) error {
	return nil
}

func (kp *Keeper) AddOrder(msg NewOrderMsg, height int64) (err error) {
	//try update order book first
	symbol := msg.Symbol
	eng, ok := kp.engines[symbol]
	if !ok {
		eng = CreateMatchEng(symbol)
		kp.engines[symbol] = eng
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

func (kp *Keeper) matchAndDistributeTrades(wg *sync.WaitGroup) []chan Transfer {
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
	for i, _ := range tradeOuts {
		tradeOuts[i] = make(chan Transfer)
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
					tradeCcy, quoteCcy, _ := utils.TradeSymbol2Ccy(ts)
					for _, t := range engine.Trades {
						t1, t2 := kp.tradeToTransfers(t, tradeCcy, quoteCcy)
						//TODO: calculate fees as transfer, f1, f2, and push into the tradeOuts
						c := channelHash(t1.account, concurrency)
						tradeOuts[c] <- t1
						c = channelHash(t2.account, concurrency)
						tradeOuts[c] <- t2
					}
					engine.DropFilledOrder()
				} // TODO: when Match() failed, have to unsolicited cancel all the orders
				// when multiple unsolicited cancel happened, the validator would stop running
				// and ask for help
				iocIDs := kp.roundIOCOrders[ts]
				for _, id := range iocIDs {
					if msg, ok := kp.allOrders[id]; ok {
						if ord, err := kp.RemoveOrder(msg.Id, msg.Symbol, msg.Side, msg.Price); err == nil {
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

func (kp *Keeper) GetOrderBookUnSafe(pair string, levelNum int, iterBuy me.LevelIter, iterSell me.LevelIter) {
	if eng, ok := kp.engines[pair]; ok {
		eng.Book.ShowDepth(levelNum, iterBuy, iterSell)
	}
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
	tradeOuts := kp.matchAndDistributeTrades(&wgOrd)
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

func (kp *Keeper) ExpireOrders(height int64, ctx sdk.Context, accountMapper auth.AccountMapper) (code sdk.CodeType, err error) {
	return sdk.CodeOK, nil
}

func (kp *Keeper) MarkBreatheBlock(height, blockTime int64, ctx sdk.Context) {
	//t := time.Unix(blockTime/1000, 0)
	//key := t.Format("20060102")
	//store := ctx.KVStore(kp.storeKey)
	//store.Set(key, height)
}

func (kp *Keeper) SnapShotOrderBook() (code sdk.CodeType, err error) {
	return sdk.CodeOK, nil
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
