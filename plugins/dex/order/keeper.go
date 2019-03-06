package order

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	dbm "github.com/tendermint/tendermint/libs/db"
	tmlog "github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/binance-chain/node/common/fees"
	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/store"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/param/paramhub"
	paramTypes "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"
)

const (
	numPricesStored  = 2000
	pricesStoreEvery = 1000
	minimalNumPrices = 500
)

type FeeHandler func(map[string]*types.Fee)
type TransferHandler func(Transfer)

// in the future, this may be distributed via Sharding
type Keeper struct {
	PairMapper                 store.TradingPairMapper
	am                         auth.AccountKeeper
	storeKey                   sdk.StoreKey // The key used to access the store from the Context.
	codespace                  sdk.CodespaceType
	engines                    map[string]*me.MatchEng
	recentPrices               map[string]*utils.FixedSizeRing  // symbol -> latest "numPricesStored" prices per "pricesStoreEvery" blocks
	allOrders                  map[string]map[string]*OrderInfo // symbol -> order ID -> order
	OrderChangesMtx            *sync.Mutex                      // guard OrderChanges and OrderInfosForPub during PreDevlierTx (which is async)
	OrderChanges               OrderChanges                     // order changed in this block, will be cleaned before matching for new block
	OrderInfosForPub           OrderInfoForPublish              // for publication usage
	roundOrders                map[string][]string              // limit to the total tx number in a block
	roundIOCOrders             map[string][]string
	RoundOrderFees             FeeHolder // order (and trade) related fee of this round, str of addr bytes -> fee
	poolSize                   uint      // number of concurrent channels, counted in the pow of 2
	cdc                        *wire.Codec
	FeeManager                 *FeeManager
	CollectOrderInfoForPublish bool
	logger                     tmlog.Logger
}

func CreateMatchEng(basePrice, lotSize int64) *me.MatchEng {
	return me.NewMatchEng(basePrice, lotSize, 0.05)
}

// NewKeeper - Returns the Keeper
func NewKeeper(key sdk.StoreKey, am auth.AccountKeeper, tradingPairMapper store.TradingPairMapper, codespace sdk.CodespaceType,
	concurrency uint, cdc *wire.Codec, collectOrderInfoForPublish bool) *Keeper {
	logger := bnclog.With("module", "dexkeeper")
	return &Keeper{
		PairMapper:                 tradingPairMapper,
		am:                         am,
		storeKey:                   key,
		codespace:                  codespace,
		engines:                    make(map[string]*me.MatchEng),
		recentPrices:               make(map[string]*utils.FixedSizeRing, 256),
		allOrders:                  make(map[string]map[string]*OrderInfo, 256), // need to init the nested map when a new symbol added.
		OrderChangesMtx:            &sync.Mutex{},
		OrderChanges:               make(OrderChanges, 0),
		OrderInfosForPub:           make(OrderInfoForPublish),
		roundOrders:                make(map[string][]string, 256),
		roundIOCOrders:             make(map[string][]string, 256),
		RoundOrderFees:             make(map[string]*types.Fee, 256),
		poolSize:                   concurrency,
		cdc:                        cdc,
		FeeManager:                 NewFeeManager(cdc, key, logger),
		CollectOrderInfoForPublish: collectOrderInfoForPublish,
		logger:                     logger,
	}
}

func (kp *Keeper) Init(ctx sdk.Context, blockInterval, daysBack int, blockDB dbm.DB, txDB dbm.DB, lastHeight int64, txDecoder sdk.TxDecoder) {
	kp.initOrderBook(ctx, blockInterval, daysBack, blockDB, txDB, lastHeight, txDecoder)
	kp.InitRecentPrices(ctx)
}

func (kp *Keeper) InitRecentPrices(ctx sdk.Context) {
	kp.recentPrices = kp.PairMapper.GetRecentPrices(ctx, pricesStoreEvery, numPricesStored)
}

func (kp *Keeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	eng := CreateMatchEng(pair.Price.ToInt64(), pair.LotSize.ToInt64())
	symbol := strings.ToUpper(pair.GetSymbol())
	kp.engines[symbol] = eng
	kp.allOrders[symbol] = map[string]*OrderInfo{}
	return eng
}

func (kp *Keeper) UpdateTickSizeAndLotSize(ctx sdk.Context) {
	tradingPairs := kp.PairMapper.ListAllTradingPairs(ctx)
	for _, pair := range tradingPairs {
		if prices, ok := kp.recentPrices[pair.GetSymbol()]; ok && prices.Count() >= minimalNumPrices {
			_, lotSize := kp.PairMapper.UpdateTickSizeAndLotSize(ctx, pair, prices)
			kp.UpdateLotSize(pair.GetSymbol(), lotSize)
		} else {
			// keep the current tick_size/lot_size
			continue
		}
	}
}

func (kp *Keeper) UpdateLotSize(symbol string, lotSize int64) {
	eng, ok := kp.engines[symbol]
	if !ok {
		panic(fmt.Sprintf("match engine of symbol %s doesn't exist", symbol))
	}
	eng.LotSize = lotSize
}

func (kp *Keeper) AddOrder(info OrderInfo, isRecovery bool) (err error) {
	//try update order book first
	symbol := strings.ToUpper(info.Symbol)
	eng, ok := kp.engines[symbol]
	if !ok {
		err = errors.New(fmt.Sprintf("match engine of symbol %s doesn't exist", symbol))
		return
	}

	_, err = eng.Book.InsertOrder(info.Id, info.Side, info.CreatedHeight, info.Price, info.Quantity)
	if err != nil {
		return err
	}

	if kp.CollectOrderInfoForPublish {
		change := OrderChange{info.Id, Ack}
		// deliberately not add this message to orderChanges
		if !isRecovery {
			kp.OrderChanges = append(kp.OrderChanges, change)
		}
		bnclog.Debug("add order to order changes map", "orderId", info.Id, "isRecovery", isRecovery)
		kp.OrderInfosForPub[info.Id] = &info
	}

	kp.allOrders[symbol][info.Id] = &info
	if ids, ok := kp.roundOrders[symbol]; ok {
		kp.roundOrders[symbol] = append(ids, info.Id)
	} else {
		newIds := make([]string, 0, 16)
		kp.roundOrders[symbol] = append(newIds, info.Id)
	}
	if info.TimeInForce == TimeInForce.IOC {
		kp.roundIOCOrders[symbol] = append(kp.roundIOCOrders[symbol], info.Id)
	}
	bnclog.Debug("Added orders", "symbol", symbol, "id", info.Id)
	return nil
}

func orderNotFound(symbol, id string) error {
	return errors.New(fmt.Sprintf("Failed to find order [%v] on symbol [%v]", id, symbol))
}

func (kp *Keeper) RemoveOrder(id string, symbol string, postCancelHandler func(ord me.OrderPart)) (err error) {
	symbol = strings.ToUpper(symbol)
	ordMsg, ok := kp.OrderExists(symbol, id)
	if !ok {
		return orderNotFound(symbol, id)
	}
	eng, ok := kp.engines[symbol]
	if !ok {
		return orderNotFound(symbol, id)
	}
	delete(kp.allOrders[symbol], id)
	ord, err := eng.Book.RemoveOrder(id, ordMsg.Side, ordMsg.Price)
	if err != nil {
		return err
	}

	if postCancelHandler != nil {
		postCancelHandler(ord)
	}
	return nil
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

func (kp *Keeper) matchAndDistributeTradesForSymbol(symbol string, height, timestamp int64, orders map[string]*OrderInfo,
	distributeTrade bool, tradeOuts []chan Transfer) {
	engine := kp.engines[symbol]
	concurrency := len(tradeOuts)
	// please note there is no logging in matching, expecting to see the order book details
	// from the exchange's order book stream.
	if engine.Match() {
		kp.logger.Debug("Match finish:", "symbol", symbol, "lastTradePrice", engine.LastTradePrice)
		for _, t := range engine.Trades {
			updateOrderMsg(orders[t.Bid], t.BuyCumQty, height, timestamp)
			updateOrderMsg(orders[t.Sid], t.SellCumQty, height, timestamp)
			if distributeTrade {
				t1, t2 := TransferFromTrade(t, symbol, kp.allOrders[symbol])
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
		kp.logger.Error("Fatal error occurred in matching, cancel all incoming new orders",
			"symbol", symbol)
		thisRoundIds := kp.roundOrders[symbol]
		for _, id := range thisRoundIds {
			msg := orders[id]
			delete(orders, id)
			if ord, err := engine.Book.RemoveOrder(id, msg.Side, msg.Price); err == nil {
				kp.logger.Info("Removed due to match failure", "ordID", msg.Id)
				if distributeTrade {
					c := channelHash(msg.Sender, concurrency)
					tradeOuts[c] <- TransferFromCanceled(ord, *msg, true)
				}
			} else {
				kp.logger.Error("Failed to remove order, may be fatal!", "orderID", id)
			}

			// let the order status publisher publish these abnormal
			// order status change outs.
			if kp.CollectOrderInfoForPublish {
				kp.OrderChangesMtx.Lock()
				kp.OrderChanges = append(kp.OrderChanges, OrderChange{id, FailedMatching})
				kp.OrderInfosForPub[id] = msg
				kp.OrderChangesMtx.Unlock()
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
				if distributeTrade {
					c := channelHash(msg.Sender, concurrency)
					tradeOuts[c] <- TransferFromExpired(ord, *msg)
				}
			} else {
				kp.logger.Error("Failed to remove IOC order, may be fatal!", "orderID", id)
			}
		}
	}
}

func (kp *Keeper) SubscribeParamChange(hub *paramhub.Keeper) {
	hub.SubscribeParamChange(
		func(ctx sdk.Context, changes []interface{}) {
			for _, c := range changes {
				switch change := c.(type) {
				case []paramTypes.FeeParam:
					feeConfig := ParamToFeeConfig(change)
					if feeConfig != nil {
						kp.FeeManager.UpdateConfig(*feeConfig)
					}
				default:
					kp.logger.Debug("Receive param changes that not interested.")
				}
			}
		},
		func(context sdk.Context, state paramTypes.GenesisState) {
			feeConfig := ParamToFeeConfig(state.FeeGenesis)
			if feeConfig != nil {
				kp.FeeManager.UpdateConfig(*feeConfig)
			} else {
				panic("Genesis with no dex fee config ")
			}
		},
		func(context sdk.Context, iLoad interface{}) {
			switch load := iLoad.(type) {
			case []paramTypes.FeeParam:
				feeConfig := ParamToFeeConfig(load)
				if feeConfig != nil {
					kp.FeeManager.UpdateConfig(*feeConfig)
				} else {
					panic("Genesis with no dex fee config ")
				}
			default:
				kp.logger.Debug("Receive param load that not interested.")
			}
		})
}

// Run as postConsume procedure of async, no concurrent updates of orders map
func updateOrderMsg(order *OrderInfo, cumQty, height, timestamp int64) {
	order.CumQty = cumQty
	order.LastUpdatedHeight = height
	order.LastUpdatedTimestamp = timestamp
}

// please note if distributeTrade this method will work in async mode, otherwise in sync mode.
func (kp *Keeper) matchAndDistributeTrades(distributeTrade bool, height, timestamp int64) []chan Transfer {
	size := len(kp.roundOrders)
	// size is the number of pairs that have new orders, i.e. it should call match()
	if size == 0 {
		kp.logger.Info("No new orders for any pair, give up matching")
		return nil
	}

	concurrency := 1 << kp.poolSize
	tradeOuts := make([]chan Transfer, concurrency)
	if distributeTrade {
		ordNum := 0
		for _, perSymbol := range kp.roundOrders {
			ordNum += len(perSymbol)
		}
		for i := range tradeOuts {
			//assume every new order would have 2 trades and generate 4 transfer
			tradeOuts[i] = make(chan Transfer, ordNum*4/concurrency)
		}
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
			kp.matchAndDistributeTradesForSymbol(symbol, height, timestamp, kp.allOrders[symbol], distributeTrade, tradeOuts)
		}
	}

	if distributeTrade {
		utils.ConcurrentExecuteAsync(concurrency, producer, matchWorker, func() {
			for _, tradeOut := range tradeOuts {
				close(tradeOut)
			}
		})
	} else {
		utils.ConcurrentExecuteSync(concurrency, producer, matchWorker)
	}
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

func (kp *Keeper) GetOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder {
	openOrders := make([]store.OpenOrder, 0)

	for _, order := range kp.allOrders[pair] {
		if string(order.Sender.Bytes()) == string(addr.Bytes()) {
			openOrders = append(
				openOrders,
				store.OpenOrder{
					order.Id,
					pair,
					utils.Fixed8(order.Price),
					utils.Fixed8(order.Quantity),
					utils.Fixed8(order.CumQty),
					order.CreatedHeight,
					order.CreatedTimestamp,
					order.LastUpdatedHeight,
					order.LastUpdatedTimestamp,
				})
		}
	}

	return openOrders
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

func (kp *Keeper) doTransfer(ctx sdk.Context, tran *Transfer) sdk.Error {
	account := kp.am.GetAccount(ctx, tran.accAddress).(types.NamedAccount)
	newLocked := account.GetLockedCoins().Minus(sdk.Coins{sdk.NewCoin(tran.outAsset, tran.unlock)})
	// these two non-negative check are to ensure the Transfer gen result is correct before we actually operate the acc.
	// they should never happen, there would be a severe bug if happen and we have to cancel all orders when app restarts.
	if !newLocked.IsNotNegative() {
		panic(errors.New(fmt.Sprintf(
			"No enough locked tokens to unlock, oid: %s, newLocked: %s, unlock: %d",
			tran.Oid,
			newLocked.String(),
			tran.unlock)))
	}
	if tran.unlock < tran.out {
		panic(errors.New("Unlocked tokens cannot cover the expense"))
	}
	account.SetLockedCoins(newLocked)
	account.SetCoins(account.GetCoins().
		Plus(sdk.Coins{sdk.NewCoin(tran.inAsset, tran.in)}).
		Plus(sdk.Coins{sdk.NewCoin(tran.outAsset, tran.unlock-tran.out)}))

	kp.am.SetAccount(ctx, account)
	kp.logger.Debug("Performed Trade Allocation", "account", account, "allocation", tran.String())
	return nil
}

func (kp *Keeper) clearAfterMatch() {
	kp.roundOrders = make(map[string][]string, 256)
	kp.roundIOCOrders = make(map[string][]string, 256)
}

func (kp *Keeper) StoreTradePrices(ctx sdk.Context) {
	// TODO: check block height != 0
	if ctx.BlockHeight()%pricesStoreEvery == 0 {
		lastTradePrices := make(map[string]int64, len(kp.engines))
		for symbol, engine := range kp.engines {
			lastTradePrices[symbol] = engine.LastTradePrice
			if _, ok := kp.recentPrices[symbol]; !ok {
				kp.recentPrices[symbol] = utils.NewFixedSizedRing(numPricesStored)
			}
			kp.recentPrices[symbol].Push(engine.LastTradePrice)
		}
		if len(lastTradePrices) != 0 {
			kp.PairMapper.UpdateRecentPrices(ctx, pricesStoreEvery, numPricesStored, lastTradePrices)
		}
	}
}

func (kp *Keeper) allocate(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer)) (
	types.Fee, map[string]*types.Fee) {
	// use string of the addr as the key since map makes a fast path for string key.
	// Also, making the key have same length is also an optimization.
	tradeInAsset := make(map[string]*sortedAsset)
	// expire fee is fixed, so we count by numbers.
	expireInAsset := make(map[string]*sortedAsset)
	// we need to distinguish different expire event, IOCExpire or Expire. only one of the two will exist.
	var expireEventType transferEventType
	var totalFee types.Fee
	for tran := range tranCh {
		kp.doTransfer(ctx, &tran)
		if !tran.FeeFree() {
			addrStr := string(tran.accAddress.Bytes())
			if tran.IsExpiredWithFee() {
				expireEventType = tran.eventType
				fees, ok := expireInAsset[addrStr]
				if !ok {
					fees = &sortedAsset{}
					expireInAsset[addrStr] = fees
				}
				fees.addAsset(tran.inAsset, 1)
			} else if tran.eventType == eventFilled {
				fees, ok := tradeInAsset[addrStr]
				if !ok {
					fees = &sortedAsset{}
					tradeInAsset[addrStr] = fees
				}
				fees.addAsset(tran.inAsset, tran.in)
			}
		}
		if postAllocateHandler != nil {
			postAllocateHandler(tran)
		}
	}

	feesPerAcc := make(map[string]*types.Fee)
	collectFee := func(assetsMap map[string]*sortedAsset, calcFeeAndDeduct func(acc sdk.Account, in sdk.Coin) types.Fee) {
		for addrStr, assets := range assetsMap {
			addr := sdk.AccAddress(addrStr)
			acc := kp.am.GetAccount(ctx, addr)
			fees := types.Fee{}
			if assets.native != 0 {
				fee := calcFeeAndDeduct(acc, sdk.NewCoin(types.NativeTokenSymbol, assets.native))
				fees.AddFee(fee)
				totalFee.AddFee(fee)
			}
			for _, asset := range assets.tokens {
				fee := calcFeeAndDeduct(acc, asset)
				fees.AddFee(fee)
				totalFee.AddFee(fee)
			}
			if !fees.IsEmpty() {
				feesPerAcc[addrStr] = &fees
				kp.am.SetAccount(ctx, acc)
			}
		}
	}
	collectFee(tradeInAsset, func(acc sdk.Account, in sdk.Coin) types.Fee {
		fee := kp.FeeManager.CalcOrderFee(acc.GetCoins(), in, kp.engines)
		acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
		return fee
	})
	collectFee(expireInAsset, func(acc sdk.Account, in sdk.Coin) types.Fee {
		var i int64 = 0
		var fees types.Fee
		for ; i < in.Amount; i++ {
			fee := kp.FeeManager.CalcFixedFee(acc.GetCoins(), expireEventType, in.Denom, kp.engines)
			acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
			fees.AddFee(fee)
		}
		return fees
	})
	return totalFee, feesPerAcc
}

func (kp *Keeper) allocateAndCalcFee(
	ctx sdk.Context,
	tradeOuts []chan Transfer,
	postAlloTransHandler TransferHandler,
) types.Fee {
	concurrency := len(tradeOuts)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	feesPerCh := make([]types.Fee, concurrency)
	feesPerAcc := make([]map[string]*types.Fee, concurrency)
	allocatePerCh := func(index int, tranCh <-chan Transfer) {
		defer wg.Done()
		fee, feeByAcc := kp.allocate(ctx, tranCh, postAlloTransHandler)
		feesPerCh[index].AddFee(fee)
		feesPerAcc[index] = feeByAcc
	}

	for i, tradeTranCh := range tradeOuts {
		go allocatePerCh(i, tradeTranCh)
	}
	wg.Wait()
	totalFee := types.Fee{}
	for i := 0; i < concurrency; i++ {
		totalFee.AddFee(feesPerCh[i])
	}
	if kp.CollectOrderInfoForPublish {
		for _, m := range feesPerAcc {
			for k, v := range m {
				kp.updateRoundOrderFee(k, *v)
			}
		}
	}
	return totalFee
}

// MatchAll will only concurrently match but do not allocate into accounts
func (kp *Keeper) MatchAll(height, timestamp int64) {
	tradeOuts := kp.matchAndDistributeTrades(false, height, timestamp) //only match
	if tradeOuts == nil {
		kp.logger.Info("No order comes in for the block")
	}
	kp.clearAfterMatch()
}

// MatchAndAllocateAll() is concurrently matching and allocating across
// all the symbols' order books, among all the clients
func (kp *Keeper) MatchAndAllocateAll(
	ctx sdk.Context,
	postAlloTransHandler TransferHandler,
) {
	bnclog.Debug("Start Matching for all...", "symbolNum", len(kp.roundOrders))
	tradeOuts := kp.matchAndDistributeTrades(true, ctx.BlockHeight(), ctx.BlockHeader().Time.Unix())
	if tradeOuts == nil {
		kp.logger.Info("No order comes in for the block")
		return
	}

	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, postAlloTransHandler)
	fees.Pool.AddAndCommitFee("MATCH", totalFee)
	kp.clearAfterMatch()
}

func (kp *Keeper) expireOrders(ctx sdk.Context, blockTime time.Time) []chan Transfer {
	size := len(kp.allOrders)
	if size == 0 {
		kp.logger.Info("No orders to expire")
		return nil
	}

	// TODO: make effectiveDays configurable
	const effectiveDays = 3
	expireHeight, err := kp.GetBreatheBlockHeight(ctx, blockTime, effectiveDays)
	if err != nil {
		// breathe block not found, that should only happens in in the first three days, just log it and ignore.
		kp.logger.Info(err.Error())
		return nil
	}

	//!!!!!!!!!!!!!!!!!!!! DELETE BEFORE MERGE
	//if ctx.BlockHeight() > 300 {
	//	expireHeight = ctx.BlockHeight() - 300
	//}
	//!!!!!!!!!!!!!!!!!!!! DELETE BEFORE MERGE

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
			transferChs[h] <- TransferFromExpired(ord, *ordMsg)
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

func (kp *Keeper) ExpireOrders(
	ctx sdk.Context,
	blockTime time.Time,
	postAlloTransHandler TransferHandler,
) {
	transferChs := kp.expireOrders(ctx, blockTime)
	if transferChs == nil {
		return
	}

	totalFee := kp.allocateAndCalcFee(ctx, transferChs, postAlloTransHandler)
	fees.Pool.AddAndCommitFee("EXPIRE", totalFee)
}

func (kp *Keeper) MarkBreatheBlock(ctx sdk.Context, height int64, blockTime time.Time) {
	key := utils.Int642Bytes(blockTime.Unix() / utils.SecondsPerDay)
	store := ctx.KVStore(kp.storeKey)
	bz, err := kp.cdc.MarshalBinaryBare(height)
	if err != nil {
		panic(err)
	}
	bnclog.Debug(fmt.Sprintf("mark breathe block for key: %v (blockTime: %d), value: %v\n", key, blockTime.Unix(), bz))
	store.Set([]byte(key), bz)
}

func (kp *Keeper) GetBreatheBlockHeight(ctx sdk.Context, timeNow time.Time, daysBack int) (int64, error) {
	store := ctx.KVStore(kp.storeKey)
	t := timeNow.AddDate(0, 0, -daysBack).Unix()
	day := t / utils.SecondsPerDay
	bz := store.Get(utils.Int642Bytes(day))
	if bz == nil {
		return 0, errors.New(fmt.Sprintf("breathe block not found for day %v", day))
	}

	var height int64
	err := kp.cdc.UnmarshalBinaryBare(bz, &height)
	if err != nil {
		panic(err)
	}
	return height, nil
}

func (kp *Keeper) GetLastBreatheBlockHeight(ctx sdk.Context, latestBlockHeight int64, timeNow time.Time, blockInterval, daysBack int) int64 {
	if blockInterval != 0 {
		return (latestBlockHeight / int64(blockInterval)) * int64(blockInterval)
	} else {
		store := ctx.KVStore(kp.storeKey)
		bz := []byte(nil)
		for i := 0; i <= daysBack; i++ {
			t := timeNow.AddDate(0, 0, -i).Unix()
			key := utils.Int642Bytes(t / utils.SecondsPerDay)
			bz = store.Get([]byte(key))
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
}

// deliberately make `fee` parameter not a pointer
// in case we modify the original fee (which will be referenced when distribute to validator)
func (kp *Keeper) updateRoundOrderFee(addr string, fee types.Fee) {
	if existingFee, ok := kp.RoundOrderFees[addr]; ok {
		existingFee.AddFee(fee)
	} else {
		kp.RoundOrderFees[addr] = &fee
	}
}

func (kp *Keeper) ClearRoundFee() {
	kp.RoundOrderFees = make(map[string]*types.Fee, 256)
}
