package order

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	tmlog "github.com/tendermint/tendermint/libs/log"

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
	allOrders                  map[string]map[string]*OrderInfo // symbol -> order ID -> order
	OrderChanges               OrderChanges                     // order changed in this block, will be cleaned before matching for new block
	OrderChangesMap            OrderInfoForPublish
	roundOrders                map[string][]string // limit to the total tx number in a block
	roundIOCOrders             map[string][]string
	roundFees                  map[string]sdk.Coins
	poolSize                   uint // number of concurrent channels, counted in the pow of 2
	cdc                        *wire.Codec
	FeeConfig                  FeeConfig
	CollectOrderInfoForPublish bool
	logger                     tmlog.Logger
}

type transferEventType uint8

const (
	eventFilled transferEventType = iota
	eventFullyExpire
	eventPartiallyExpire
	eventIOCFullyExpire
	eventIOCPartiallyExpire
	eventExpireForMatchFailure
)

// Transfer represents a transfer between trade currencies
type Transfer struct {
	Oid        string
	eventType  transferEventType
	accAddress sdk.AccAddress
	inAsset    string
	in         int64
	outAsset   string
	out        int64
	unlock     int64
	Fee        types.Fee
	Trade      *me.Trade
	Symbol     string
}

func (tran Transfer) FeeFree() bool {
	return tran.eventType == eventPartiallyExpire ||
		tran.eventType == eventIOCPartiallyExpire || tran.eventType == eventExpireForMatchFailure
}

func (tran Transfer) IsExpire() bool {
	return tran.eventType == eventIOCFullyExpire || tran.eventType == eventIOCPartiallyExpire || tran.eventType == eventPartiallyExpire || tran.eventType == eventFullyExpire
}

func (tran Transfer) IsExpiredWithFee() bool {
	return tran.eventType == eventFullyExpire || tran.eventType == eventIOCFullyExpire
}

func (tran *Transfer) String() string {
	return fmt.Sprintf("Transfer[eventType:%v, oid:%v, inAsset:%v, inQty:%v, outAsset:%v, outQty:%v, unlock:%v, fee:%v]",
		tran.eventType, tran.Oid, tran.inAsset, tran.in, tran.outAsset, tran.out, tran.unlock, tran.Fee)
}

func CreateMatchEng(lotSize int64) *me.MatchEng {
	return me.NewMatchEng(1000, lotSize, 0.05)
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
		allOrders:                  make(map[string]map[string]*OrderInfo, 256), // need to init the nested map when a new symbol added.
		OrderChanges:               make(OrderChanges, 0),
		OrderChangesMap:            make(OrderInfoForPublish),
		roundOrders:                make(map[string][]string, 256),
		roundIOCOrders:             make(map[string][]string, 256),
		poolSize:                   concurrency,
		cdc:                        cdc,
		FeeConfig:                  NewFeeConfig(cdc, key),
		CollectOrderInfoForPublish: collectOrderInfoForPublish,
		logger: bnclog.With("module", "dexkeeper"),
	}
}

func (kp *Keeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	eng := CreateMatchEng(pair.Price.ToInt64(), pair.LotSize.ToInt64())
	symbol := strings.ToUpper(pair.GetSymbol())
	kp.engines[symbol] = eng
	kp.allOrders[symbol] = map[string]*OrderInfo{}
	return eng
}

func (kp *Keeper) UpdateLotSize(symbol string, lotSize int64) {
	eng, ok := kp.engines[symbol]
	if !ok {
		panic(fmt.Sprintf("match engine of symbol %s doesn't exist", symbol))
	}
	eng.LotSize = lotSize
}

func (kp *Keeper) AddOrder(msg OrderInfo, height int64, isRecovery bool) (err error) {
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
		change := OrderChange{msg.Id, Ack, 0, ""}
		// deliberately not add this message to orderChanges
		if !isRecovery {
			kp.OrderChanges = append(kp.OrderChanges, change)
		}
		bnclog.Debug("add order to order changes map", "orderId", msg.Id, "isRecovery", isRecovery)
		kp.OrderChangesMap[msg.Id] = &msg
	}

	kp.allOrders[symbol][msg.Id] = &msg
	if ids, ok := kp.roundOrders[symbol]; ok {
		kp.roundOrders[symbol] = append(ids, msg.Id)
	} else {
		newIds := make([]string, 0, 16)
		kp.roundOrders[symbol] = append(newIds, msg.Id)
	}
	if msg.TimeInForce == TimeInForce.IOC {
		kp.roundIOCOrders[symbol] = append(kp.roundIOCOrders[symbol], msg.Id)
	}
	bnclog.Debug("Added orders", "symbol", symbol, "id", msg.Id)
	return nil
}

func orderNotFound(symbol, id string) error {
	return errors.New(fmt.Sprintf("Failed to find order [%v] on symbol [%v]", id, symbol))
}

func (kp *Keeper) RemoveOrder(
	id string,
	symbol string,
	side int8,
	price int64,
	isRecovery bool) (ord me.OrderPart, err error) {
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
	if kp.CollectOrderInfoForPublish && isRecovery {
		bnclog.Debug("deleted order from order changes map", "orderId", msg.Id, "isRecovery", isRecovery)
		delete(kp.OrderChangesMap, msg.Id) // for nonRecovery, will remove during endblock
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

func (kp *Keeper) OrderExists(symbol, id string) (OrderInfo, bool) {
	if orders, ok := kp.allOrders[symbol]; ok {
		if msg, ok := orders[id]; ok {
			return *msg, ok
		}
	}
	return OrderInfo{}, false
}

func (kp *Keeper) tradeToTransfers(trade me.Trade, symbol string) (Transfer, Transfer) {
	baseAsset, quoteAsset, _ := utils.TradingPair2Assets(symbol)
	seller := kp.allOrders[symbol][trade.Sid].Sender
	buyer := kp.allOrders[symbol][trade.Bid].Sender
	// TODO: where is 10^8 stored?
	quoteQty := utils.CalBigNotional(trade.LastPx, trade.LastQty)
	unlock := utils.CalBigNotional(trade.OrigBuyPx, trade.BuyCumQty) - utils.CalBigNotional(trade.OrigBuyPx, trade.BuyCumQty-trade.LastQty)
	return Transfer{trade.Sid, eventFilled, seller, quoteAsset, quoteQty, baseAsset, trade.LastQty, trade.LastQty, types.Fee{}, &trade, symbol},
		Transfer{trade.Bid, eventFilled, buyer, baseAsset, trade.LastQty, quoteAsset, quoteQty, unlock, types.Fee{}, &trade, symbol}
}

func (kp *Keeper) expiredToTransfer(ord me.OrderPart, ordMsg *OrderInfo, tranEventType transferEventType) Transfer {
	//here is a trick to use the same currency as in and out ccy to simulate cancel
	qty := ord.LeavesQty()
	baseAsset, quoteAsset, _ := utils.TradingPair2Assets(ordMsg.Symbol)
	var unlock int64
	var unlockAsset string
	if ordMsg.Side == Side.BUY {
		unlockAsset = quoteAsset
		unlock = utils.CalBigNotional(ordMsg.Price, ordMsg.Quantity) - utils.CalBigNotional(ordMsg.Price, ordMsg.Quantity-qty)
	} else {
		unlockAsset = baseAsset
		unlock = qty
	}

	if ord.CumQty != 0 && tranEventType != eventExpireForMatchFailure {
		if ordMsg.TimeInForce == TimeInForce.IOC {
			tranEventType = eventIOCPartiallyExpire // IOC partially filled
		} else {
			tranEventType = eventPartiallyExpire
		}
	}

	return Transfer{
		Oid:        ordMsg.Id,
		eventType:  tranEventType,
		accAddress: ordMsg.Sender,
		inAsset:    unlockAsset,
		in:         unlock,
		outAsset:   unlockAsset,
		out:        unlock,
		unlock:     unlock,
	}
}

// channelHash() will choose a channel for processing by moding
// the sum of the last 7 bytes of address by bucketNumber.
// It may not be fully even.
// TODO: there is still concern on peroformance and evenness.
func channelHash(accAddress sdk.AccAddress, bucketNumber int) int {
	l := len(accAddress)
	sum := 0
	for i := l - 7; i < l; i++ {
		sum += int(accAddress[i])
	}
	return sum % bucketNumber
}

func (kp *Keeper) matchAndDistributeTradesForSymbol(symbol string, orders map[string]*OrderInfo,
	distributeTrade bool, tradeOuts []chan Transfer) {
	engine := kp.engines[symbol]
	concurrency := len(tradeOuts)
	// please note there is no logging in matching, expecting to see the order book details
	// from the exchange's order book stream.
	if engine.Match() {
		kp.logger.Debug("Match finish:", "symbol", symbol, "lastTradePrice", engine.LastTradePrice)
		for _, t := range engine.Trades {
			orders[t.Bid].CumQty = t.BuyCumQty
			orders[t.Sid].CumQty = t.SellCumQty

			if distributeTrade {
				t1, t2 := kp.tradeToTransfers(t, symbol)
				c := channelHash(t1.accAddress, concurrency)
				tradeOuts[c] <- t1
				c = channelHash(t2.accAddress, concurrency)
				tradeOuts[c] <- t2
			}
		}
		droppedIds := engine.DropFilledOrder() //delete from order books
		for _, id := range droppedIds {
			delete(orders, id) //delete from order cache
		}
		kp.logger.Debug("Drop filled orders", "total", droppedIds)
	} else {
		// FUTURE-TODO:
		// when Match() failed, have to unsolicited cancel all the new orders
		// in this block. Ideally the order IDs would be stored in the EndBlock response,
		// but this is not implemented yet, pending Tendermint to better handle EndBlock
		// for index service.
		//
		// the order status publisher should publish these abnormal
		// order status change out too.
		kp.logger.Error("Fatal error occurred in matching, cancell all incoming new orders",
			"symbol", symbol)
		thisRoundIds := kp.roundOrders[symbol]
		for _, id := range thisRoundIds {
			msg := orders[id]
			delete(orders, id)
			if ord, err := engine.Book.RemoveOrder(id, msg.Side, msg.Price); err == nil {
				kp.logger.Info("Removed due to match failure", "ordID", msg.Id)
				if !distributeTrade {
					continue
				}
				c := channelHash(msg.Sender, concurrency)
				tradeOuts[c] <- kp.expiredToTransfer(ord, msg, eventExpireForMatchFailure)
			} else {
				kp.logger.Error("Failed to remove order, may be fatal!", "orderID", id)
			}
		}
		return // no need to handle IOC
	}
	iocIDs := kp.roundIOCOrders[symbol]
	for _, id := range iocIDs {
		if msg, ok := orders[id]; ok {
			delete(orders, id)
			if ord, err := engine.Book.RemoveOrder(id, msg.Side, msg.Price); err == nil {
				kp.logger.Debug("Removed unclosed IOC order", "ordID", msg.Id)
				if !distributeTrade {
					continue
				}
				c := channelHash(msg.Sender, concurrency)
				//cumQty would be tested inside expiredToTransfer
				tradeOuts[c] <- kp.expiredToTransfer(ord, msg, eventIOCFullyExpire)
			} else {
				kp.logger.Error("Failed to remove IOC order, may be fatal!", "orderID", id)
			}
		}
	}
}

func (kp *Keeper) matchAndDistributeTrades(distributeTrade bool) []chan Transfer {
	size := len(kp.roundOrders)
	// size is the number of pairs that have new orders, i.e. it should call match()
	if size == 0 {
		kp.logger.Info("No new orders for any pair, give up matching")
		return nil
	}

	ordNum := 0
	for _, perSymbol := range kp.roundOrders {
		ordNum += len(perSymbol)
	}
	concurrency := 1 << kp.poolSize

	tradeOuts := make([]chan Transfer, concurrency)
	if !distributeTrade {
		ordNum = 0
	}
	for i := range tradeOuts {
		//assume every new order would have 2 trades and generate 4 transfer
		tradeOuts[i] = make(chan Transfer, ordNum*4/concurrency)
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
			kp.logger.Debug("start matching", "symbol", symbol)
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

func (kp *Keeper) GetOrderBooks(maxLevels int) ChangedPriceLevelsMap {
	var res = make(ChangedPriceLevelsMap)
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

func (kp *Keeper) GetTradeAndOrdersRelatedAccounts(orders []OrderChange) []string {
	res := make([]string, 0)

	for _, eng := range kp.engines {
		for _, t := range eng.Trades {
			if orderChange, exists := kp.OrderChangesMap[t.Bid]; exists {
				res = append(res, string(orderChange.Sender.Bytes()))
			} else {
				bnclog.Error("fail to locate order in order changes map", "orderId", t.Bid)
			}
			if orderChange, exists := kp.OrderChangesMap[t.Sid]; exists {
				res = append(res, string(orderChange.Sender.Bytes()))
			} else {
				bnclog.Error("fail to locate order in order changes map", "orderId", t.Sid)
			}
		}
	}

	for _, orderChange := range orders {
		res = append(res, string(kp.OrderChangesMap[orderChange.Id].Sender.Bytes()))
	}

	return res
}

func (kp *Keeper) GetLastTradesForPair(pair string) ([]me.Trade, int64) {
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

func (kp *Keeper) ClearOrderChanges() {
	kp.OrderChanges = kp.OrderChanges[:0]
}

func (kp *Keeper) doTransfer(ctx sdk.Context, am auth.AccountMapper, tran *Transfer) sdk.Error {
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
	kp.logger.Debug("Performed Trade Allocation", "account", account, "allocation", tran.String())
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
			kp.logger.Debug("Not enough native token to pay trade fee", "feeToken", feeToken)
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
		kp.logger.Error("Invalid expire eventType", "eventType", tran.eventType)
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

func (kp *Keeper) clearAfterMatch() {
	kp.roundOrders = make(map[string][]string, 256)
	kp.roundIOCOrders = make(map[string][]string, 256)
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
		kp.logger.Info("No order comes in for the block")
		return sdk.CodeOK, nil
	}

	// the following code is to wait for all match finished.
	var wg sync.WaitGroup
	wg.Add(len(tradeOuts))
	concurrentSettle(&wg, tradeOuts, func(int, Transfer) {})
	wg.Wait()
	kp.clearAfterMatch()
	return sdk.CodeOK, nil
}

// MatchAndAllocateAll() is concurrently matching and allocating across
// all the symbols' order books, among all the clients
// TODO: the return value: code & err may not be required.
func (kp *Keeper) MatchAndAllocateAll(ctx sdk.Context, am auth.AccountMapper,
	postAllocateHandler func(tran Transfer)) (newCtx sdk.Context, code sdk.CodeType, err error) {
	bnclog.Debug("Start Matching for all...", "symbolNum", len(kp.roundOrders))
	tradeOuts := kp.matchAndDistributeTrades(true)
	if tradeOuts == nil {
		kp.logger.Info("No order comes in for the block")
		return ctx, sdk.CodeOK, nil
	}

	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, am, postAllocateHandler)
	newCtx = tx.WithFee(ctx, totalFee)
	kp.clearAfterMatch()
	return newCtx, sdk.CodeOK, nil
}

func (kp *Keeper) expireOrders(ctx sdk.Context, blockTime int64, am auth.AccountMapper) []chan Transfer {
	size := len(kp.allOrders)
	if size == 0 {
		kp.logger.Info("No orders to expire")
		return nil
	}

	// TODO: make effectiveDays configurable
	const effectiveDays = 3
	expireHeight, err := kp.GetBreatheBlockHeight(ctx, time.Unix(blockTime, 0), effectiveDays)
	if err != nil {
		// breathe block not found, that should only happens in in the first three days, just log it and ignore.
		kp.logger.Info(err.Error())
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

	expire := func(orders map[string]*OrderInfo, engine *me.MatchEng, side int8) {
		engine.Book.RemoveOrders(expireHeight, side, func(ord me.OrderPart) {
			// gen transfer
			ordMsg := orders[ord.Id]
			h := channelHash(ordMsg.Sender, concurrency)
			//cumQty would be tested inside expiredToTransfer
			transferChs[h] <- kp.expiredToTransfer(ord, ordMsg, eventFullyExpire)
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
	for i := 0; i <= daysBack; i++ {
		t := timeNow.AddDate(0, 0, -i).Unix()
		key := utils.Int642Bytes(t / SecondsInOneDay)
		bz = kvStore.Get([]byte(key))
		if bz != nil {
			kp.logger.Info("Located day to load breathe block height", "epochDay", key)
			break
		}
	}
	if bz == nil {
		kp.logger.Error("Failed to load the latest breathe block height from", "timeNow", timeNow)
		return 0
	}
	var height int64
	err := kp.cdc.UnmarshalBinaryBare(bz, &height)
	if err != nil {
		panic(err)
	}
	kp.logger.Info("Loaded breathe block height", "height", height)
	return height
}

func (kp *Keeper) InitGenesis(ctx sdk.Context, genesis TradingGenesis) {
	kp.logger.Info("Initializing Fees from Genesis")
	kp.FeeConfig.InitGenesis(ctx, genesis)
}
