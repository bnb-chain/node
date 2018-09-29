package order

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/pkg/errors"

	bnclog "github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	dexTypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

// in the future, this may be distributed via Sharding
type Keeper struct {
	PairMapper store.TradingPairMapper

	ck bank.Keeper

	storeKey                   sdk.StoreKey // The key used to access the store from the Context.
	codespace                  sdk.CodespaceType
	engines                    map[string]*me.MatchEng
	allOrders                  map[string]map[string]NewOrderMsg // symbol -> order ID -> order
	OrderChanges               OrderChanges                      // order changed in this block, will be cleaned before matching for new block
	OrderChangesMap            OrderChangesMap
	roundOrders                map[string]int // limit to the total tx number in a block
	roundIOCOrders             map[string][]string
	roundFees                  map[string]sdk.Coins
	poolSize                   uint // number of concurrent channels, counted in the pow of 2
	cdc                        *wire.Codec
	FeeConfig                  FeeConfig
	CollectOrderInfoForPublish bool
}

type transferEventType uint8

const (
	eventFilled transferEventType = iota
	eventFullyExpire
	eventPartiallyExpire
	eventIOCFullyExpire
	eventIOCPartiallyExpire
)

// Transfer represents a transfer between trade currencies
type Transfer struct {
	Bid        string
	Sid        string
	eventType  transferEventType
	accAddress sdk.AccAddress
	inAsset    string
	in         int64
	outAsset   string
	out        int64
	unlock     int64
	Fee        types.Fee
}

func (tran Transfer) IsBuyer() bool {
	return strings.HasPrefix(tran.Bid, tran.accAddress.String())
}

func (tran Transfer) GetPairSymbol() string {
	if tran.IsBuyer() {
		return fmt.Sprintf("%s_%s", tran.inAsset, tran.outAsset)
	} else {
		return fmt.Sprintf("%s_%s", tran.outAsset, tran.inAsset)
	}
}

func (tran Transfer) FeeFree() bool {
	return tran.eventType == eventPartiallyExpire || tran.eventType == eventIOCPartiallyExpire
}

func (tran Transfer) IsExpiredWithFee() bool {
	return tran.eventType == eventFullyExpire || tran.eventType == eventIOCFullyExpire
}

func (tran *Transfer) String() string {
	return fmt.Sprintf("Transfer[eventType:%v, bid:%v, sid:%v, inAsset:%v, inQty:%v, outAsset:%v, outQty:%v, unlock:%v, fee:%v]",
		tran.eventType, tran.Bid, tran.Sid, tran.inAsset, tran.in, tran.outAsset, tran.out, tran.unlock, tran.Fee)
}

func CreateMatchEng(basePrice, lotSize int64) *me.MatchEng {
	return me.NewMatchEng(basePrice, lotSize, 0.05)
}

// NewKeeper - Returns the Keeper
func NewKeeper(key sdk.StoreKey, bankKeeper bank.Keeper, tradingPairMapper store.TradingPairMapper, codespace sdk.CodespaceType,
	concurrency uint, cdc *wire.Codec, collectOrderInfoForPublish bool) *Keeper {
	engines := make(map[string]*me.MatchEng)
	return &Keeper{
		PairMapper:                 tradingPairMapper,
		ck:                         bankKeeper,
		storeKey:                   key,
		codespace:                  codespace,
		engines:                    engines,
		allOrders:                  make(map[string]map[string]NewOrderMsg, 256), // need to init the nested map when a new symbol added.
		OrderChanges:               make(OrderChanges, 0),
		OrderChangesMap:            make(OrderChangesMap),
		roundOrders:                make(map[string]int, 256),
		roundIOCOrders:             make(map[string][]string, 256),
		poolSize:                   concurrency,
		cdc:                        cdc,
		FeeConfig:                  NewFeeConfig(cdc, key),
		CollectOrderInfoForPublish: collectOrderInfoForPublish,
	}
}

func (kp *Keeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	eng := CreateMatchEng(pair.Price.ToInt64(), pair.LotSize.ToInt64())
	symbol := strings.ToUpper(pair.GetSymbol())
	kp.engines[symbol] = eng
	kp.allOrders[symbol] = map[string]NewOrderMsg{}
	return eng
}

func (kp *Keeper) UpdateLotSize(symbol string, lotSize int64) {
	eng, ok := kp.engines[symbol]
	if !ok {
		panic(fmt.Sprintf("match engine of symbol %s doesn't exist", symbol))
	}
	eng.LotSize = lotSize
}

func (kp *Keeper) AddOrder(msg NewOrderMsg, height int64, txHash string, isReplay bool) (err error) {
	//try update order book first
	symbol := strings.ToUpper(msg.Symbol)
	eng, ok := kp.engines[symbol]
	if !ok {
		err = errors.New(fmt.Sprintf("match engine of symbol %s doesn't exist", symbol))
		return
	}

	_, err = eng.Book.InsertOrder(msg.Id, msg.Side, height, msg.Price, msg.Quantity)
	if err != nil {
		return err
	}

	if kp.CollectOrderInfoForPublish {
		change := OrderChange{OrderMsg: msg, TxHash: txHash, Tpe: Ack, Fee: 0}
		// deliberately not add this message to orderChanges
		if !isReplay {
			kp.OrderChanges = append(kp.OrderChanges, change)
		}
		bnclog.Debug(fmt.Sprintf("add order to order changes map", "orderId", msg.Id, "isReplay", isReplay))
		kp.OrderChangesMap[msg.Id] = &change
	}

	kp.allOrders[symbol][msg.Id] = msg
	kp.roundOrders[symbol] += 1
	if msg.TimeInForce == TimeInForce.IOC {
		kp.roundIOCOrders[symbol] = append(kp.roundIOCOrders[symbol], msg.Id)
	}
	return nil
}

func orderNotFound(symbol, id string) error {
	return errors.New(fmt.Sprintf("Failed to find order [%v] on symbol [%v]", id, symbol))
}

// txHash is empty string for reason except for Cancel
func (kp *Keeper) RemoveOrder(id string, symbol string, side int8, price int64, txHash string, reason ChangeType, isReplay bool) (ord me.OrderPart, err error) {
	symbol = strings.ToUpper(symbol)
	msg, ok := kp.OrderExists(symbol, id)
	if !ok {
		return me.OrderPart{}, orderNotFound(symbol, id)
	}
	eng, ok := kp.engines[symbol]
	if !ok {
		return me.OrderPart{}, orderNotFound(symbol, id)
	}
	delete(kp.allOrders[symbol], id)
	ord, err = eng.Book.RemoveOrder(id, side, price)
	if kp.CollectOrderInfoForPublish && !isReplay {
		// fee will be updated during doTransfer
		change := OrderChange{OrderMsg: msg, Tpe: reason, Fee: 0, CumQty: ord.CumQty, TxHash: txHash}
		kp.OrderChanges = append(kp.OrderChanges, change)
	}
	return ord, err
}

func (kp *Keeper) GetOrder(id string, symbol string, side int8, price int64) (ord me.OrderPart, err error) {
	symbol = strings.ToUpper(symbol)
	_, ok := kp.OrderExists(symbol, id)
	if !ok {
		return me.OrderPart{}, orderNotFound(symbol, id)
	}
	eng, ok := kp.engines[symbol]
	if !ok {
		return me.OrderPart{}, orderNotFound(symbol, id)
	}
	return eng.Book.GetOrder(id, side, price)
}

func (kp *Keeper) OrderExists(symbol, id string) (NewOrderMsg, bool) {
	if orders, ok := kp.allOrders[symbol]; ok {
		if msg, ok := orders[id]; ok {
			return msg, ok
		}
	}
	return NewOrderMsg{}, false
}

func (kp *Keeper) tradeToTransfers(trade me.Trade, symbol string) (Transfer, Transfer) {
	baseAsset, quoteAsset, _ := utils.TradingPair2Assets(symbol)
	seller := kp.allOrders[symbol][trade.SId].Sender
	buyer := kp.allOrders[symbol][trade.BId].Sender
	// TODO: where is 10^8 stored?
	quoteQty := utils.CalBigNotional(trade.LastPx, trade.LastQty)
	unlock := utils.CalBigNotional(trade.OrigBuyPx, trade.BuyCumQty) - utils.CalBigNotional(trade.OrigBuyPx, trade.BuyCumQty-trade.LastQty)
	return Transfer{trade.BId, trade.SId, eventFilled, seller, quoteAsset, quoteQty, baseAsset, trade.LastQty, trade.LastQty, types.Fee{}},
		Transfer{trade.BId, trade.SId, eventFilled, buyer, baseAsset, trade.LastQty, quoteAsset, quoteQty, unlock, types.Fee{}}
}

func (kp *Keeper) expiredToTransfer(ord me.OrderPart, ordMsg NewOrderMsg) Transfer {
	//here is a trick to use the same currency as in and out ccy to simulate cancel
	qty := ord.LeavesQty()
	baseAsset, quoteAsset, _ := utils.TradingPair2Assets(ordMsg.Symbol)
	var unlock int64
	var unlockAsset string
	var bid string
	var sid string
	if ordMsg.Side == Side.BUY {
		bid = ordMsg.Id
		unlockAsset = quoteAsset
		unlock = utils.CalBigNotional(ordMsg.Price, ordMsg.Quantity) - utils.CalBigNotional(ordMsg.Price, ordMsg.Quantity-qty)
	} else {
		sid = ordMsg.Id
		unlockAsset = baseAsset
		unlock = qty
	}

	var tranEventType transferEventType
	if ord.CumQty == 0 {
		if ordMsg.TimeInForce == TimeInForce.IOC {
			tranEventType = eventIOCFullyExpire // IOC no fill
		} else {
			tranEventType = eventFullyExpire
		}
	} else {
		if ordMsg.TimeInForce == TimeInForce.IOC {
			tranEventType = eventIOCPartiallyExpire // IOC partially filled
		} else {
			tranEventType = eventPartiallyExpire
		}
	}
	return Transfer{
		Bid:        bid,
		Sid:        sid,
		eventType:  tranEventType,
		accAddress: ordMsg.Sender,
		inAsset:    unlockAsset,
		in:         unlock,
		outAsset:   unlockAsset,
		out:        unlock,
		unlock:     unlock,
	}
}

//TODO: should get an even hash
func channelHash(accAddress sdk.AccAddress, bucketNumber int) int {
	return int(accAddress[0]+accAddress[1]) % bucketNumber
}

func (kp *Keeper) matchAndDistributeTradesForSymbol(symbol string, orders map[string]NewOrderMsg, distributeTrade bool,
	tradeOuts []chan Transfer) {
	engine := kp.engines[symbol]
	concurrency := len(tradeOuts)
	logger := bnclog.With("module", "dex")
	// please note there is no logging in matching, expecting to see the order book details
	// from the exchange's order book stream.
	if engine.Match() {
		logger.Debug("Match finish:", "symbol", symbol, "lastTradePrice", engine.LastTradePrice)
		if distributeTrade {
			for _, t := range engine.Trades {
				t1, t2 := kp.tradeToTransfers(t, symbol)
				c := channelHash(t1.accAddress, concurrency)
				tradeOuts[c] <- t1
				c = channelHash(t2.accAddress, concurrency)
				tradeOuts[c] <- t2
			}
		}
		n := engine.DropFilledOrder()
		logger.Debug("Drop filled orders", "total", n)
	} // TODO: when Match() failed, have to unsolicited cancel all the orders
	// when multiple unsolicited cancel happened, the validator would stop running
	// and ask for help
	iocIDs := kp.roundIOCOrders[symbol]
	for _, id := range iocIDs {
		if msg, ok := orders[id]; ok {
			if ord, err := kp.RemoveOrder(msg.Id, msg.Symbol, msg.Side, msg.Price, "", IocNoFill, false); err == nil {
				logger.Debug("Removed unclosed IOC order", "ordID", msg.Id)
				if !distributeTrade {
					continue
				}
				c := channelHash(msg.Sender, concurrency)
				tradeOuts[c] <- kp.expiredToTransfer(ord, msg)
			}
		}
	}
}

func (kp *Keeper) matchAndDistributeTrades(distributeTrade bool) []chan Transfer {
	size := len(kp.roundOrders)
	// size is the number of pairs that have new orders, i.e. it should call match()
	if size == 0 {
		return nil
	}
	channelSize := size >> kp.poolSize
	concurrency := 1 << kp.poolSize
	if size%concurrency != 0 {
		channelSize += 1
	}

	tradeOuts := make([]chan Transfer, concurrency)
	for i := range tradeOuts {
		// TODO: channelSize is enough for buffer to facilitate ?
		if distributeTrade {
			channelSize = 0
		}
		tradeOuts[i] = make(chan Transfer, channelSize*2)
	}

	symbolCh := make(chan string, concurrency)
	producer := func() {
		for symbol := range kp.roundOrders {
			symbolCh <- symbol
		}
		close(symbolCh)
	}
	matchWorker := func() {
		for symbol := range symbolCh {
			kp.matchAndDistributeTradesForSymbol(symbol, kp.allOrders[symbol], distributeTrade, tradeOuts)
		}
	}
	utils.ConcurrentExecuteAsync(concurrency, producer, matchWorker, func() {
		for _, tradeOut := range tradeOuts {
			close(tradeOut)
		}
	})

	return tradeOuts
}

func (kp *Keeper) GetOrderBookLevels(pair string, maxLevels int) []store.OrderBookLevel {
	orderbook := make([]store.OrderBookLevel, maxLevels)

	i, j := 0, 0

	if eng, ok := kp.engines[pair]; ok {
		// TODO: check considered bucket splitting?
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

func (kp *Keeper) GetOrderBookForPublish(maxLevels int) ChangedPriceLevels {
	var res = make(ChangedPriceLevels)
	for pair, eng := range kp.engines {
		buys := make(map[int64]int64)
		sells := make(map[int64]int64)
		res[pair] = ChangedPriceLevelsPerSymbol{buys, sells}

		// TODO: check considered bucket splitting?
		eng.Book.ShowDepth(maxLevels, func(p *me.PriceLevel) {
			buys[p.Price] = p.TotalLeavesQty()
		}, func(p *me.PriceLevel) {
			sells[p.Price] = p.TotalLeavesQty()
		})
	}
	return res
}

func (kp *Keeper) GetLastTrades() *map[string][]me.Trade {
	resT := make(map[string][]me.Trade, len(kp.engines))

	for pair := range kp.engines {
		trades, _ := kp.GetLastTradesForPair(pair)
		resT[pair] = make([]me.Trade, len(trades))
		for idx, trade := range trades {
			resT[pair][idx] = trade
		}
	}

	return &resT
}

func (kp *Keeper) GetTradeRelatedAccounts(orders []OrderChange) *[]string {
	res := make([]string, 0)

	for pair := range kp.engines {
		trades, _ := kp.GetLastTradesForPair(pair)
		for _, t := range trades {
			if orderChange, exists := kp.OrderChangesMap[t.BId]; exists {
				res = append(res, string(orderChange.OrderMsg.Sender.Bytes()))
			} else {
				bnclog.Error(fmt.Sprintf("fail to know locate order %s in order changes map", t.BId))
			}
			if orderChange, exists := kp.OrderChangesMap[t.SId]; exists {
				res = append(res, string(orderChange.OrderMsg.Sender.Bytes()))
			} else {
				bnclog.Error(fmt.Sprintf("fail to know locate order %s in order changes map", t.SId))
			}
		}
	}

	for _, orderChange := range orders {
		res = append(res, string(orderChange.OrderMsg.Sender.Bytes()))
	}

	return &res
}

func (kp *Keeper) GetLastTradesForPair(pair string) ([]me.Trade, int64) {
	if eng, ok := kp.engines[pair]; ok {
		return eng.Trades, eng.LastTradePrice
	}
	return nil, 0
}

func (kp *Keeper) GetLastOrdersCopy() (OrderChanges, OrderChangesMap) {
	var orderChangesSnapshot = make(OrderChanges, len(kp.OrderChanges))
	copy(orderChangesSnapshot, kp.OrderChanges)

	var orderChangesMapSnapshot = make(OrderChangesMap, len(kp.OrderChangesMap))
	for k, v := range kp.OrderChangesMap {
		orderChangesMapSnapshot[k] = v
	}

	return orderChangesSnapshot, orderChangesMapSnapshot
}

func (kp *Keeper) ClearOrderBook(pair string) {
	if eng, ok := kp.engines[pair]; ok {
		eng.Book.Clear()
	}
}

func (kp *Keeper) ClearOrderChanges() {
	kp.OrderChanges = kp.OrderChanges[:0]
}

func (kp *Keeper) doTransfer(ctx sdk.Context, am auth.AccountMapper, tran *Transfer) sdk.Error {
	logger := bnclog.With("module", "dex")
	account := am.GetAccount(ctx, tran.accAddress).(types.NamedAccount)
	newLocked := account.GetLockedCoins().Minus(sdk.Coins{sdk.NewCoin(tran.outAsset, tran.unlock)})
	if !newLocked.IsNotNegative() {
		return sdk.ErrInternal("No enough locked tokens to unlock")
	}
	account.SetLockedCoins(newLocked)
	account.SetCoins(account.GetCoins().
		Plus(sdk.Coins{sdk.NewCoin(tran.inAsset, tran.in)}).
		Plus(sdk.Coins{sdk.NewCoin(tran.outAsset, tran.unlock-tran.out)}))

	if !tran.FeeFree() {
		fee := kp.calcFeeFromTransfer(ctx, account, *tran)
		if !fee.IsEmpty() {
			account.SetCoins(account.GetCoins().Minus(fee.Tokens.Sort()))
		}
		tran.Fee = fee
	}
	am.SetAccount(ctx, account)
	logger.Debug("Performed Trade Allocation", "account", account, "allocation", tran.String())
	return nil
}

func (kp *Keeper) calcFeeFromTransfer(ctx sdk.Context, account auth.Account, tran Transfer) types.Fee {
	if tran.eventType == eventFilled {
		return kp.calcOrderFee(ctx, account, tran)
	} else if tran.eventType == eventFullyExpire || tran.eventType == eventIOCFullyExpire {
		return kp.calcExpireFee(ctx, tran)
	}

	return types.Fee{}
}

func (kp *Keeper) calcOrderFee(ctx sdk.Context, account auth.Account, tran Transfer) types.Fee {
	var feeToken sdk.Coin
	if tran.inAsset == types.NativeToken {
		feeToken = sdk.NewCoin(types.NativeToken, kp.FeeConfig.CalcFee(tran.in, FeeByNativeToken))
	} else {
		// price against native token
		var amountOfNativeToken int64
		if engine, ok := kp.engines[utils.Assets2TradingPair(tran.inAsset, types.NativeToken)]; ok {
			// XYZ_BNB
			amountOfNativeToken = utils.CalBigNotional(engine.LastTradePrice, tran.in)
		} else {
			// BNB_XYZ
			price := kp.engines[utils.Assets2TradingPair(types.NativeToken, tran.inAsset)].LastTradePrice
			var amount big.Int
			amountOfNativeToken = amount.Div(amount.Mul(big.NewInt(tran.in), big.NewInt(utils.Fixed8One.ToInt64())), big.NewInt(price)).Int64()
		}
		feeByNativeToken := kp.FeeConfig.CalcFee(amountOfNativeToken, FeeByNativeToken)
		if account.GetCoins().AmountOf(types.NativeToken).Int64() >= feeByNativeToken {
			// have sufficient native token to pay the fees
			feeToken = sdk.NewCoin(types.NativeToken, feeByNativeToken)
		} else {
			// no enough NativeToken, use the received tokens as fee
			feeToken = sdk.NewCoin(tran.inAsset, kp.FeeConfig.CalcFee(tran.in, FeeByTradeToken))
			logger := bnclog.With("module", "dex")
			logger.Debug("Not enough native token to pay trade fee", "feeToken", feeToken)
		}
	}

	return types.NewFee(sdk.Coins{feeToken}, types.FeeForProposer)
}

func (kp *Keeper) calcExpireFee(ctx sdk.Context, tran Transfer) types.Fee {
	var feeAmount int64
	if tran.eventType == eventFullyExpire {
		feeAmount = kp.FeeConfig.ExpireFee()
	} else if tran.eventType == eventIOCFullyExpire {
		feeAmount = kp.FeeConfig.IOCExpireFee()
	} else {
		// should not be here
		bnclog.With("module", "dex").Error("Invalid expire eventType", "eventType", tran.eventType)
		return types.Fee{}
	}

	// in a Transfer of expire event type, inAsset == outAsset, in == out == unlock
	// to make the calc logic consistent with calcOrderFee, we always use in/inAsset to calc the fee.
	if tran.inAsset != types.NativeToken {
		if engine, ok := kp.engines[utils.Assets2TradingPair(tran.inAsset, types.NativeToken)]; ok {
			// XYZ_BNB
			var amount big.Int
			feeAmount = amount.Div(
				amount.Mul(big.NewInt(feeAmount), big.NewInt(utils.Fixed8One.ToInt64())),
				big.NewInt(engine.LastTradePrice)).Int64()
		} else {
			// BNB_XYZ
			engine = kp.engines[utils.Assets2TradingPair(types.NativeToken, tran.inAsset)]
			feeAmount = utils.CalBigNotional(engine.LastTradePrice, feeAmount)
		}
	}

	if tran.in < feeAmount {
		feeAmount = tran.in
	}
	return types.NewFee(sdk.Coins{sdk.NewCoin(tran.inAsset, feeAmount)}, types.FeeForProposer)
}

func (kp *Keeper) clearAfterMatch() (err error) {
	kp.roundOrders = make(map[string]int, 256)
	kp.roundIOCOrders = make(map[string][]string, 256)
	return nil
}

func concurrentSettle(wg *sync.WaitGroup, tradeOuts []chan Transfer, settleHandler func(int, Transfer)) {
	for i, tradeTranCh := range tradeOuts {
		go func(index int, tranCh <-chan Transfer) {
			defer wg.Done()
			for tran := range tranCh {
				settleHandler(index, tran)
			}
		}(i, tradeTranCh)
	}
}

func (kp *Keeper) allocateAndCalcFee(ctx sdk.Context, tradeOuts []chan Transfer, am auth.AccountMapper, postAllocateHandler func(tran Transfer)) types.Fee {
	concurrency := len(tradeOuts)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	feesPerCh := make([]types.Fee, concurrency)
	allocate := func(index int, tran Transfer) {
		kp.doTransfer(ctx, am, &tran)
		feesPerCh[index].AddFee(tran.Fee)
		if postAllocateHandler != nil {
			postAllocateHandler(tran)
		}
	}
	concurrentSettle(&wg, tradeOuts, allocate)
	wg.Wait()
	totalFee := tx.Fee(ctx)
	for i := 0; i < concurrency; i++ {
		totalFee.AddFee(feesPerCh[i])
	}
	return totalFee
}

// MatchAll will only concurrently match but do not allocate into accounts
func (kp *Keeper) MatchAll() (code sdk.CodeType, err error) {
	tradeOuts := kp.matchAndDistributeTrades(false) //only match
	if tradeOuts == nil {
		bnclog.With("module", "dex").Info("No order comes in for the block")
		return sdk.CodeOK, nil
	}

	// the following code is to wait for all match finished.
	var wg sync.WaitGroup
	wg.Add(len(tradeOuts))
	concurrentSettle(&wg, tradeOuts, func(int, Transfer) {})
	wg.Wait()
	return sdk.CodeOK, nil
}

// MatchAndAllocateAll() is concurrently matching and allocating across
// all the symbols' order books, among all the clients
func (kp *Keeper) MatchAndAllocateAll(ctx sdk.Context, am auth.AccountMapper,
	postAllocateHandler func(tran Transfer)) (newCtx sdk.Context, code sdk.CodeType, err error) {
	tradeOuts := kp.matchAndDistributeTrades(true)
	if tradeOuts == nil {
		bnclog.With("module", "dex").Info("No order comes in for the block")
		return ctx, sdk.CodeOK, nil
	}

	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, am, postAllocateHandler)
	newCtx = tx.WithFee(ctx, totalFee)
	return newCtx, sdk.CodeOK, nil
}

func (kp *Keeper) expireOrders(ctx sdk.Context, blockTime int64, am auth.AccountMapper) []chan Transfer {
	logger := bnclog.With("module", "dex")
	size := len(kp.allOrders)
	if size == 0 {
		logger.Info("No orders to expire")
		return nil
	}

	// TODO: make effectiveDays configurable
	const effectiveDays = 3
	expireHeight, err := kp.GetBreatheBlockHeight(ctx, time.Unix(blockTime, 0), effectiveDays)
	if err != nil {
		// breathe block not found, that should only happens in in the first three days, just log it and ignore.
		logger.Info(err.Error())
		return nil
	}

	channelSize := size >> kp.poolSize
	concurrency := 1 << kp.poolSize
	if size%concurrency != 0 {
		channelSize += 1
	}

	transferChs := make([]chan Transfer, concurrency)
	for i := range transferChs {
		// TODO: channelSize is enough for buffer to facilitate ?
		transferChs[i] = make(chan Transfer, channelSize*2)
	}

	expire := func(orders map[string]NewOrderMsg, engine *me.MatchEng, side int8) {
		engine.Book.RemoveOrders(expireHeight, side, func(ord me.OrderPart) {
			// gen transfer
			ordMsg := orders[ord.Id]
			h := channelHash(ordMsg.Sender, concurrency)
			transferChs[h] <- kp.expiredToTransfer(ord, ordMsg)
			// delete from allOrders
			delete(orders, ord.Id)
		})
	}

	symbolCh := make(chan string, concurrency)
	utils.ConcurrentExecuteAsync(concurrency,
		func() {
			for symbol, _ := range kp.allOrders {
				symbolCh <- symbol
			}
			close(symbolCh)
		}, func() {
			for symbol := range symbolCh {
				engine := kp.engines[symbol]
				orders := kp.allOrders[symbol]
				expire(orders, engine, me.BUYSIDE)
				expire(orders, engine, me.SELLSIDE)
			}
		}, func() {
			for _, transferCh := range transferChs {
				close(transferCh)
			}
		})

	return transferChs
}

func (kp *Keeper) ExpireOrders(ctx sdk.Context, blockTime int64, am auth.AccountMapper, postExpireHandler func(Transfer)) (newCtx sdk.Context, code sdk.CodeType, err error) {
	transferChs := kp.expireOrders(ctx, blockTime, am)
	if transferChs == nil {
		return ctx, sdk.CodeOK, nil
	}

	totalFee := kp.allocateAndCalcFee(ctx, transferChs, am, postExpireHandler)
	newCtx = tx.WithFee(ctx, totalFee)
	return newCtx, sdk.CodeOK, nil
}

func (kp *Keeper) MarkBreatheBlock(ctx sdk.Context, height, blockTime int64) {
	key := utils.Int642Bytes(blockTime / utils.SecondsPerDay)
	store := ctx.KVStore(kp.storeKey)
	bz, err := kp.cdc.MarshalBinaryBare(height)
	if err != nil {
		panic(err)
	}
	bnclog.Debug(fmt.Sprintf("mark breathe block for key: %v (blockTime: %d), value: %v\n", key, blockTime, bz))
	store.Set([]byte(key), bz)
}

func (kp *Keeper) GetBreatheBlockHeight(ctx sdk.Context, timeNow time.Time, daysBack int) (int64, error) {
	store := ctx.KVStore(kp.storeKey)
	t := timeNow.AddDate(0, 0, -daysBack).Unix()
	day := t / utils.SecondsPerDay
	bz := store.Get(utils.Int642Bytes(day))
	if bz == nil {
		return 0, errors.Errorf("breathe block not found for day %v", day)
	}

	var height int64
	err := kp.cdc.UnmarshalBinaryBare(bz, &height)
	if err != nil {
		panic(err)
	}
	return height, nil
}

func (kp *Keeper) getLastBreatheBlockHeight(ctx sdk.Context, timeNow time.Time, daysBack int) int64 {
	store := ctx.KVStore(kp.storeKey)
	bz := []byte(nil)
	logger := bnclog.With("module", "dex")
	for i := 0; i <= daysBack; i++ {
		t := timeNow.AddDate(0, 0, -i).Unix()
		key := utils.Int642Bytes(t / utils.SecondsPerDay)
		bz = store.Get([]byte(key))
		if bz != nil {
			logger.Info("Located day to load breathe block height", "epochDay", key)
			break
		}
	}
	if bz == nil {
		logger.Error("Failed to load the latest breathe block height from", "timeNow", timeNow)
		return 0
	}
	var height int64
	err := kp.cdc.UnmarshalBinaryBare(bz, &height)
	if err != nil {
		panic(err)
	}
	logger.Info("Loaded breathe block height", "height", height)
	return height
}

func (kp *Keeper) InitGenesis(ctx sdk.Context, genesis TradingGenesis) {
	bnclog.With("module", "dex").Info("Initializing Fees from Genesis")
	kp.FeeConfig.InitGenesis(ctx, genesis)
}
