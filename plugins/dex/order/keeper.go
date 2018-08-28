package order

import (
	"errors"
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/tx"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	dexTypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

const SecondsInOneDay = 24 * 60 * 60

// in the future, this may be distributed via Sharding
type Keeper struct {
	PairMapper store.TradingPairMapper

	ck bank.Keeper

	storeKey       sdk.StoreKey // The key used to access the store from the Context.
	codespace      sdk.CodespaceType
	engines        map[string]*me.MatchEng
	allOrders      map[string]NewOrderMsg // symbol -> order ID -> order
	roundOrders    map[string]int         // limit to the total tx number in a block
	roundIOCOrders map[string][]string
	roundFees      map[string]sdk.Coins
	poolSize       uint // number of concurrent channels, counted in the pow of 2
	cdc            *wire.Codec
	FeeConfig      FeeConfig
}

type transferEventType int64

const (
	eventFilled = iota
	eventFullyExpire
	eventPartiallyExpire
	eventIocFullyExpire
	eventIocPartiallyExpire
)

// Transfer represents a transfer between trade currencies
type Transfer struct {
	bid        string
	sid        string
	eventType  transferEventType
	accAddress sdk.AccAddress
	inCcy      string
	in         int64
	outCcy     string
	out        int64
	unlock     int64
	fee        types.Fee
}

func (tran Transfer) feeFree() bool {
	return tran.eventType == eventPartiallyExpire || tran.eventType == eventIocPartiallyExpire
}

func CreateMatchEng(lotSize int64) *me.MatchEng {
	return me.NewMatchEng(1000, lotSize, 0.05)
}

// NewKeeper - Returns the Keeper
func NewKeeper(key sdk.StoreKey, bankKeeper bank.Keeper, tradingPairMapper store.TradingPairMapper, codespace sdk.CodespaceType,
	concurrency uint, cdc *wire.Codec) *Keeper {
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
		FeeConfig:      NewFeeConfig(key),
	}
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
	return Transfer{trade.BId, trade.SId, eventFilled, seller, quoteCcy, quoteQty, tradeCcy, trade.LastQty, trade.LastQty, types.Fee{}},
		Transfer{trade.BId, trade.SId, eventFilled, buyer, tradeCcy, trade.LastQty, quoteCcy, quoteQty, unlock, types.Fee{}}
}

//TODO: should get an even hash
func channelHash(accAddress sdk.AccAddress, bucketNumber int) int {
	return int(accAddress[0]+accAddress[1]) % bucketNumber
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
	index := 0
	for k := range kp.roundOrders {
		outs[index/channelSize][index%channelSize] = k
		index++
	}
	tradeOuts := make([]chan Transfer, concurrency)
	if distributeTrade {
		for i, _ := range tradeOuts {
			//TODO: channelSize is enough for buffer to facilitate ?
			tradeOuts[i] = make(chan Transfer, channelSize)
		}
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
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
							c := channelHash(t1.accAddress, concurrency)
							tradeOuts[c] <- t1
							c = channelHash(t2.accAddress, concurrency)
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

							var tranEventType transferEventType
							if ord.CumQty == 0 {
								// IOC no fill
								tranEventType = eventIocFullyExpire
							} else {
								// IOC partially filled
								tranEventType = eventIocPartiallyExpire
							}
							tradeOuts[c] <- Transfer{
								eventType:  tranEventType,
								accAddress: msg.Sender,
								inCcy:      tradeCcy,
								in:         qty,
								outCcy:     tradeCcy,
								out:        qty,
								unlock:     unlock,
							}
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

func (kp *Keeper) doTransfer(ctx sdk.Context, accountMapper auth.AccountMapper, tran *Transfer) sdk.Error {
	account := accountMapper.GetAccount(ctx, tran.accAddress).(types.NamedAccount)
	newLocked := account.GetLockedCoins().Minus(sdk.Coins{sdk.Coin{Denom: tran.outCcy, Amount: sdk.NewInt(tran.unlock)}})
	if !newLocked.IsNotNegative() {
		return sdk.ErrInternal("No enough locked tokens to unlock")
	}
	account.SetLockedCoins(newLocked)
	account.SetCoins(account.GetCoins().Plus(sdk.Coins{
		sdk.Coin{Denom: tran.inCcy, Amount: sdk.NewInt(tran.in)},
		sdk.Coin{Denom: tran.outCcy, Amount: sdk.NewInt(tran.unlock - tran.out)}}.Sort()))

	if !tran.feeFree() {
		var fee types.Fee
		if tran.eventType == eventFilled {
			fee = kp.calculateOrderFee(ctx, account, *tran)
			account.SetCoins(account.GetCoins().Minus(fee.Tokens))
		} else if tran.eventType == eventFullyExpire {
			//
		} else if tran.eventType == eventIocFullyExpire {
			//
		}

		tran.fee = fee
	}
	accountMapper.SetAccount(ctx, account)
	return nil
}

func (kp *Keeper) calculateOrderFee(ctx sdk.Context, account auth.Account, tran Transfer) types.Fee {
	var feeToken sdk.Coin
	if tran.inCcy == types.NativeToken {
		feeToken = sdk.NewCoin(types.NativeToken, calcFee(tran.in, kp.FeeConfig.feeRateWithNativeToken))
	} else {
		symbol := utils.Ccy2TradeSymbol(tran.inCcy, types.NativeToken)
		// price against native token
		price := kp.engines[symbol].LastTradePrice
		feeByNativeToken := calcFee(utils.CalBigNotional(price, tran.in), kp.FeeConfig.feeRateWithNativeToken)
		if account.GetCoins().AmountOf(types.NativeToken).Int64() >= feeByNativeToken {
			// have sufficient native token to pay the fees
			feeToken = sdk.NewCoin(types.NativeToken, feeByNativeToken)
		} else {
			// no enough NativeToken, use the received tokens as fee
			feeToken = sdk.NewCoin(tran.inCcy, calcFee(tran.in, kp.FeeConfig.feeRate))
		}
	}

	return types.NewFee(sdk.Coins{feeToken}, types.FeeForProposer)
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
func (kp *Keeper) MatchAndAllocateAll(ctx sdk.Context, accountMapper auth.AccountMapper) (newCtx sdk.Context, code sdk.CodeType, err error) {
	var wg sync.WaitGroup
	allocate := func(ctx sdk.Context, accountMapper auth.AccountMapper, transChan <-chan Transfer, settled chan<- Transfer) {
		defer wg.Done()
		for tran := range transChan {
			kp.doTransfer(ctx, accountMapper, &tran)
			settled <- tran
		}
	}
	var wgOrd sync.WaitGroup
	tradeOuts := kp.matchAndDistributeTrades(&wgOrd, true)
	if tradeOuts == nil {
		//TODO: logging
		return ctx, sdk.CodeOK, nil
	}

	wg.Add(len(tradeOuts))
	settledChan := make(chan Transfer, len(tradeOuts)*2)
	for _, tran := range tradeOuts {
		go allocate(ctx, accountMapper, tran, settledChan)
	}

	settleDone := make(chan struct{})
	go func() {
		defer close(settleDone)
		settledList := make([]Transfer, 0)
		totalFee := tx.Fee(ctx)
		for settled := range settledChan {
			settledList = append(settledList, settled)
			totalFee.AddFee(settled.fee)
		}
		// WithSettlement should only be called once in each block.
		newCtx = WithSettlement(tx.WithFee(ctx, totalFee), settledList)
	}()

	wgOrd.Wait()
	for _, t := range tradeOuts {
		close(t)
	}
	wg.Wait()
	close(settledChan)
	<-settleDone
	return newCtx, sdk.CodeOK, nil
}

func (kp *Keeper) ExpireOrders(ctx sdk.Context, height int64, accountMapper auth.AccountMapper) (code sdk.CodeType, err error) {
	return sdk.CodeOK, nil
}

func (kp *Keeper) MarkBreatheBlock(ctx sdk.Context, height, blockTime int64) {
	key := utils.Int642Bytes(blockTime / SecondsInOneDay)
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
		key := utils.Int642Bytes(t / SecondsInOneDay)
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

func (kp *Keeper) InitGenesis(ctx sdk.Context, genesis TradingGenesis) {
	kp.FeeConfig.InitGenesis(ctx, genesis)
}
