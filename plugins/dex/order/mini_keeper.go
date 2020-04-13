package order

import (
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/binance-chain/node/common/fees"
	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/store"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/wire"
)

const (
	defaultMiniBlockMatchInterval = 16
	defaultActiveMiniSymbolCount  = 8
)

//order keeper for mini-token
type MiniKeeper struct {
	Keeper                               //use dex order keeper as base keeper
	matchedMiniSymbols []string          //mini token pairs matched in this round
	miniSymbolsHash    map[string]uint32 //mini token pairs -> hash value for Round-Robin
}

var _ DexOrderKeeper = &MiniKeeper{}

// NewKeeper - Returns the MiniToken Keeper
func NewMiniKeeper(dexMiniKey sdk.StoreKey, am auth.AccountKeeper, miniPairMapper store.TradingPairMapper, codespace sdk.CodespaceType,
	concurrency uint, cdc *wire.Codec, collectOrderInfoForPublish bool) *MiniKeeper {
	logger := bnclog.With("module", "dexkeeper")
	return &MiniKeeper{
		Keeper{PairMapper: miniPairMapper,
			am:                         am,
			storeKey:                   dexMiniKey,
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
			FeeManager:                 NewFeeManager(cdc, dexMiniKey, logger),
			CollectOrderInfoForPublish: collectOrderInfoForPublish,
			logger:                     logger},
		make([]string, 0, 256),
		make(map[string]uint32, 256),
	}
}

// override
func (kp *MiniKeeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	eng := kp.Keeper.AddEngine(pair)
	symbol := strings.ToUpper(pair.GetSymbol())
	kp.miniSymbolsHash[symbol] = crc32.ChecksumIEEE([]byte(symbol))
	return eng
}

// override
// please note if distributeTrade this method will work in async mode, otherwise in sync mode.
func (kp *MiniKeeper) matchAndDistributeTrades(distributeTrade bool, height, timestamp int64, matchAllMiniSymbols bool) ([]chan Transfer) {
	size := len(kp.roundOrders)
	// size is the number of pairs that have new orders, i.e. it should call match()
	if size == 0 {
		kp.logger.Info("No new orders for any pair, give up matching")
		return nil
	}

	concurrency := 1 << kp.poolSize
	tradeOuts := make([]chan Transfer, concurrency)

	if matchAllMiniSymbols {
		for symbol := range kp.roundOrders {
			kp.matchedMiniSymbols = append(kp.matchedMiniSymbols, symbol)
		}
	} else {
		kp.selectMiniSymbolsToMatch(height, func(miniSymbols map[string]struct{}) {
			for symbol := range miniSymbols {
				kp.matchedMiniSymbols = append(kp.matchedMiniSymbols, symbol)
			}
		})
	}

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
		i := 0
		for symbol := range symbolCh {
			i++
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

func (kp *MiniKeeper) GetOrderBookLevels(pair string, maxLevels int) []store.OrderBookLevel {
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

func (kp *MiniKeeper) GetOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder {
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

func (kp *MiniKeeper) GetOrderBooks(maxLevels int) ChangedPriceLevelsMap {
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

func (kp *MiniKeeper) GetPriceLevel(pair string, side int8, price int64) *me.PriceLevel {
	if eng, ok := kp.engines[pair]; ok {
		return eng.Book.GetPriceLevel(price, side)
	} else {
		return nil
	}
}

func (kp *MiniKeeper) GetLastTrades(height int64, pair string) ([]me.Trade, int64) {
	if eng, ok := kp.engines[pair]; ok {
		if eng.LastMatchHeight == height {
			return eng.Trades, eng.LastTradePrice
		}
	}
	return nil, 0
}

// !!! FOR TEST USE ONLY
func (kp *MiniKeeper) GetLastTradesForPair(pair string) ([]me.Trade, int64) {
	if eng, ok := kp.engines[pair]; ok {
		return eng.Trades, eng.LastTradePrice
	}
	return nil, 0
}

func (kp *MiniKeeper) ClearOrderBook(pair string) {
	if eng, ok := kp.engines[pair]; ok {
		eng.Book.Clear()
	}
}

func (kp *MiniKeeper) ClearOrderChanges() {
	kp.OrderChanges = kp.OrderChanges[:0]
}

func (kp *MiniKeeper) doTransfer(ctx sdk.Context, tran *Transfer) sdk.Error {
	account := kp.am.GetAccount(ctx, tran.accAddress).(types.NamedAccount)
	newLocked := account.GetLockedCoins().Minus(sdk.Coins{sdk.NewCoin(tran.outAsset, tran.unlock)})
	// these two non-negative check are to ensure the Transfer gen result is correct before we actually operate the acc.
	// they should never happen, there would be a severe bug if happen and we have to cancel all orders when app restarts.
	if !newLocked.IsNotNegative() {
		panic(fmt.Errorf(
			"no enough locked tokens to unlock, oid: %s, newLocked: %s, unlock: %d",
			tran.Oid,
			newLocked.String(),
			tran.unlock))
	}
	if tran.unlock < tran.out {
		panic(errors.New("unlocked tokens cannot cover the expense"))
	}
	account.SetLockedCoins(newLocked)
	accountCoin := account.GetCoins().
		Plus(sdk.Coins{sdk.NewCoin(tran.inAsset, tran.in)})
	if remain := tran.unlock - tran.out; remain > 0 || !sdk.IsUpgrade(upgrade.FixZeroBalance) {
		accountCoin = accountCoin.Plus(sdk.Coins{sdk.NewCoin(tran.outAsset, remain)})
	}
	account.SetCoins(accountCoin)

	kp.am.SetAccount(ctx, account)
	return nil
}

// override
func (kp *MiniKeeper) clearAfterMatch() {
	for _, symbol := range kp.matchedMiniSymbols {
		delete(kp.roundOrders, symbol)
		delete(kp.roundIOCOrders, symbol)
	}
	kp.matchedMiniSymbols = make([]string, 0, 256)
}

func (kp *MiniKeeper) StoreTradePrices(ctx sdk.Context) {
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

func (kp *MiniKeeper) allocate(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer)) (
	types.Fee, map[string]*types.Fee) {
	if !sdk.IsUpgrade(upgrade.BEP19) {
		return kp.allocateBeforeGalileo(ctx, tranCh, postAllocateHandler)
	}

	// use string of the addr as the key since map makes a fast path for string key.
	// Also, making the key have same length is also an optimization.
	tradeTransfers := make(map[string]TradeTransfers)
	// expire fee is fixed, so we count by numbers.
	expireTransfers := make(map[string]ExpireTransfers)
	// we need to distinguish different expire event, IOCExpire or Expire. only one of the two will exist.
	var expireEventType transferEventType
	var totalFee types.Fee
	for tran := range tranCh {
		kp.doTransfer(ctx, &tran)
		if !tran.FeeFree() {
			addrStr := string(tran.accAddress.Bytes())
			// need a copy of tran as it is reused
			tranCp := tran
			if tran.IsExpiredWithFee() {
				expireEventType = tran.eventType
				if _, ok := expireTransfers[addrStr]; !ok {
					expireTransfers[addrStr] = ExpireTransfers{&tranCp}
				} else {
					expireTransfers[addrStr] = append(expireTransfers[addrStr], &tranCp)
				}
			} else if tran.eventType == eventFilled {
				if _, ok := tradeTransfers[addrStr]; !ok {
					tradeTransfers[addrStr] = TradeTransfers{&tranCp}
				} else {
					tradeTransfers[addrStr] = append(tradeTransfers[addrStr], &tranCp)
				}
			}
		} else if tran.IsExpire() {
			if postAllocateHandler != nil {
				postAllocateHandler(tran)
			}
		}
	}

	feesPerAcc := make(map[string]*types.Fee)
	for addrStr, trans := range tradeTransfers {
		addr := sdk.AccAddress(addrStr)
		acc := kp.am.GetAccount(ctx, addr)
		fees := kp.FeeManager.CalcTradesFee(acc.GetCoins(), trans, kp.engines)
		if !fees.IsEmpty() {
			feesPerAcc[addrStr] = &fees
			acc.SetCoins(acc.GetCoins().Minus(fees.Tokens))
			kp.am.SetAccount(ctx, acc)
			totalFee.AddFee(fees)
		}
	}

	for addrStr, trans := range expireTransfers {
		addr := sdk.AccAddress(addrStr)
		acc := kp.am.GetAccount(ctx, addr)

		fees := kp.FeeManager.CalcExpiresFee(acc.GetCoins(), expireEventType, trans, kp.engines, postAllocateHandler)
		if !fees.IsEmpty() {
			if _, ok := feesPerAcc[addrStr]; ok {
				feesPerAcc[addrStr].AddFee(fees)
			} else {
				feesPerAcc[addrStr] = &fees
			}
			acc.SetCoins(acc.GetCoins().Minus(fees.Tokens))
			kp.am.SetAccount(ctx, acc)
			totalFee.AddFee(fees)
		}
	}
	return totalFee, feesPerAcc
}

// DEPRECATED
func (kp *MiniKeeper) allocateBeforeGalileo(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer)) (
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
				// no possible to overflow, for tran.in == otherSide.tran.out <= TotalSupply(otherSide.tran.outAsset)
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

			var fees types.Fee
			if exists, ok := feesPerAcc[addrStr]; ok {
				fees = *exists
			}
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
		fee := kp.FeeManager.CalcTradeFee(acc.GetCoins(), in, kp.engines)
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

func (kp *MiniKeeper) allocateAndCalcFee(
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
func (kp *MiniKeeper) MatchAll(height, timestamp int64) {
	tradeOuts := kp.matchAndDistributeTrades(false, height, timestamp, false) //only match
	if tradeOuts == nil {
		kp.logger.Info("No order comes in for the block")
	}
	kp.clearAfterMatch()
}

// MatchAndAllocateAll() is concurrently matching and allocating across
// all the symbols' order books, among all the clients
// Return whether match has been done in this height
func (kp *MiniKeeper) MatchAndAllocateAll(ctx sdk.Context, postAlloTransHandler TransferHandler, matchAllSymbols bool) {
	kp.logger.Debug("Start Matching for all...", "height", ctx.BlockHeader().Height, "symbolNum", len(kp.roundOrders))
	timestamp := ctx.BlockHeader().Time.UnixNano()
	tradeOuts := kp.matchAndDistributeTrades(true, ctx.BlockHeader().Height, timestamp, matchAllSymbols)
	if tradeOuts == nil {
		kp.logger.Info("No order comes in for the block")
	}

	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, postAlloTransHandler)
	fees.Pool.AddAndCommitFee("MATCH", totalFee)
	kp.clearAfterMatch()
}

func (kp *MiniKeeper) expireOrders(ctx sdk.Context, blockTime time.Time) []chan Transfer {
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
			if ordMsg, ok := orders[ord.Id]; ok && ordMsg != nil {
				h := channelHash(ordMsg.Sender, concurrency)
				transferChs[h] <- TransferFromExpired(ord, *ordMsg)
				// delete from allOrders
				delete(orders, ord.Id)
			} else {
				kp.logger.Error("failed to locate order to remove in order book", "oid", ord.Id)
			}
		})
	}

	symbolCh := make(chan string, concurrency)
	utils.ConcurrentExecuteAsync(concurrency,
		func() {
			for symbol := range kp.allOrders {
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

func (kp *MiniKeeper) ExpireOrders(
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

// used by state sync to clear memory order book after we synced latest breathe block
//TODO check usage
func (kp *MiniKeeper) ClearOrders() {
	kp.Keeper.ClearOrders()
	kp.matchedMiniSymbols = make([]string, 0, 256)
}

func (kp *MiniKeeper) DelistTradingPair(ctx sdk.Context, symbol string, postAllocTransHandler TransferHandler) {
	_, ok := kp.engines[symbol]
	if !ok {
		kp.logger.Error("delist symbol does not exist", "symbol", symbol)
		return
	}

	transferChs := kp.expireAllOrders(ctx, symbol)
	if transferChs != nil {
		totalFee := kp.allocateAndCalcFee(ctx, transferChs, postAllocTransHandler)
		fees.Pool.AddAndCommitFee(fmt.Sprintf("DELIST_%s", symbol), totalFee)
	}

	delete(kp.engines, symbol)
	delete(kp.allOrders, symbol)
	delete(kp.recentPrices, symbol)

	baseAsset, quoteAsset := dexUtils.TradingPair2AssetsSafe(symbol)
	err := kp.PairMapper.DeleteTradingPair(ctx, baseAsset, quoteAsset)
	if err != nil {
		kp.logger.Error("delete trading pair error", "err", err.Error())
	}
}

//override
func (kp *MiniKeeper) CanListTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error {
	// trading pair against native token should exist if quote token is not native token
	baseAsset = strings.ToUpper(baseAsset)
	quoteAsset = strings.ToUpper(quoteAsset)

	if baseAsset == quoteAsset {
		return fmt.Errorf("base asset symbol should not be identical to quote asset symbol")
	}

	if kp.PairMapper.Exists(ctx, baseAsset, quoteAsset) || kp.PairMapper.Exists(ctx, quoteAsset, baseAsset) {
		return errors.New("trading pair exists")
	}

	if types.NativeTokenSymbol != quoteAsset { //todo permit BUSD
		return errors.New("quote token is not valid: " + quoteAsset)
	}

	return nil
}

//override TODO check
func (kp *MiniKeeper) CanDelistTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error {
	// trading pair against native token should not be delisted if there is any other trading pair exist
	baseAsset = strings.ToUpper(baseAsset)
	quoteAsset = strings.ToUpper(quoteAsset)

	if baseAsset == quoteAsset {
		return fmt.Errorf("base asset symbol should not be identical to quote asset symbol")
	}

	if !kp.PairMapper.Exists(ctx, baseAsset, quoteAsset) {
		return fmt.Errorf("trading pair %s_%s does not exist", baseAsset, quoteAsset)
	}

	return nil
}

func (kp *MiniKeeper) selectMiniSymbolsToMatch(height int64, postSelect func(map[string]struct{})) {
	symbolsToMatch := make(map[string]struct{}, 256)
	selectActiveMiniSymbols(&symbolsToMatch, &kp.roundOrders, defaultActiveMiniSymbolCount)
	selectMiniSymbolsRoundRobin(&symbolsToMatch, &kp.miniSymbolsHash, height)
	postSelect(symbolsToMatch)
}

func selectActiveMiniSymbols(symbolsToMatch *map[string]struct{}, roundOrdersMini *map[string][]string, k int) {
	//use quick select to select top k symbols
	symbolOrderNumsSlice := make([]*SymbolWithOrderNumber, 0, len(*roundOrdersMini))
	i := 0
	for symbol, orders := range *roundOrdersMini {
		symbolOrderNumsSlice[i] = &SymbolWithOrderNumber{symbol, len(orders)}
	}
	topKSymbolOrderNums := findTopKLargest(symbolOrderNumsSlice, k)

	for _, selected := range topKSymbolOrderNums {
		(*symbolsToMatch)[selected.symbol] = struct{}{}
	}
}

func selectMiniSymbolsRoundRobin(symbolsToMatch *map[string]struct{}, miniSymbolsHash *map[string]uint32, height int64) {
	m := height % defaultMiniBlockMatchInterval
	for symbol, symbolHash := range *miniSymbolsHash {
		if int64(symbolHash%defaultMiniBlockMatchInterval) == m {
			(*symbolsToMatch)[symbol] = struct{}{}
		}
	}
}

// override
func (kp *MiniKeeper) validateOrder(ctx sdk.Context, acc sdk.Account, msg NewOrderMsg) error {

	err := kp.Keeper.validateOrder(ctx, acc, msg)
	if err != nil {
		return err
	}
	coins := acc.GetCoins()
	symbol := strings.ToUpper(msg.Symbol)
	var quantityBigEnough bool
	if msg.Side == Side.BUY {
		quantityBigEnough = msg.Quantity >= types.MiniTokenMinTotalSupply
	} else if msg.Side == Side.SELL {
		quantityBigEnough = (msg.Quantity >= types.MiniTokenMinTotalSupply) || coins.AmountOf(symbol) == msg.Quantity
	}
	if !quantityBigEnough {
		return fmt.Errorf("quantity is too small, the min quantity is %d or total free balance of the mini token",
			types.MiniTokenMinTotalSupply)
	}
	return nil
}
