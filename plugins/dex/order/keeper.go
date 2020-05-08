package order

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/fees"
	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/matcheng"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/store"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/plugins/param/paramhub"
	paramTypes "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"
)

const (
	BEP2TypeValue = 1
	MiniTypeValue = 2
)

type SymbolPairType int8

var PairType = struct {
	BEP2 SymbolPairType
	MINI SymbolPairType
}{BEP2TypeValue, MiniTypeValue}

type IDexKeeper interface {
	InitRecentPrices(ctx sdk.Context)
	AddEngine(pair dexTypes.TradingPair) *me.MatchEng
	UpdateTickSizeAndLotSize(ctx sdk.Context)
	DetermineLotSize(baseAssetSymbol, quoteAssetSymbol string, price int64) (lotSize int64)
	UpdateLotSize(symbol string, lotSize int64)
	AddOrder(info OrderInfo, isRecovery bool) (err error)
	RemoveOrder(id string, symbol string, postCancelHandler func(ord me.OrderPart)) (err error)
	GetOrder(id string, symbol string, side int8, price int64) (ord me.OrderPart, err error)
	OrderExists(symbol, id string) (OrderInfo, bool)
	GetOrderBookLevels(pair string, maxLevels int) []store.OrderBookLevel
	GetOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder
	GetOrderBooks(maxLevels int, pairType SymbolPairType) ChangedPriceLevelsMap
	GetPriceLevel(pair string, side int8, price int64) *me.PriceLevel
	GetLastTrades(height int64, pair string) ([]me.Trade, int64)
	GetLastTradesForPair(pair string) ([]me.Trade, int64)
	ClearOrderBook(pair string)
	ClearOrderChanges()
	StoreTradePrices(ctx sdk.Context)
	ExpireOrders(ctx sdk.Context, blockTime time.Time, postAlloTransHandler TransferHandler)
	MarkBreatheBlock(ctx sdk.Context, height int64, blockTime time.Time)
	GetBreatheBlockHeight(ctx sdk.Context, timeNow time.Time, daysBack int) (int64, error)
	GetLastBreatheBlockHeight(ctx sdk.Context, latestBlockHeight int64, timeNow time.Time, blockInterval, daysBack int) int64
	DelistTradingPair(ctx sdk.Context, symbol string, postAllocTransHandler TransferHandler)
	CanListTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error
	CanDelistTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error
	SnapShotOrderBook(ctx sdk.Context, height int64) (effectedStoreKeys []string, err error)
	LoadOrderBookSnapshot(ctx sdk.Context, latestBlockHeight int64, timeOfLatestBlock time.Time, blockInterval, daysBack int) (int64, error)
	GetPairMapper() store.TradingPairMapper
	GetOrderChanges(pairType SymbolPairType) OrderChanges
	GetOrderInfosForPub(pairType SymbolPairType) OrderInfoForPublish
	GetAllOrders() map[string]map[string]*OrderInfo
	GetAllOrdersForPair(symbol string) map[string]*OrderInfo
	getAccountKeeper() *auth.AccountKeeper
	getLogger() tmlog.Logger
	getFeeManager() *FeeManager
	GetEngines() map[string]*me.MatchEng
	ShouldPublishOrder() bool
	UpdateOrderChange(change OrderChange, symbol string)
	UpdateOrderChangeSync(change OrderChange, symbol string)
	ValidateOrder(context sdk.Context, account sdk.Account, msg NewOrderMsg) error
	SelectSymbolsToMatch(height, timestamp int64, matchAllSymbols bool) []string
	ReloadOrder(symbol string, orderInfo *OrderInfo, height int64)
}

type DexKeeper struct {
	PairMapper                 store.TradingPairMapper
	storeKey                   sdk.StoreKey // The key used to access the store from the Context.
	codespace                  sdk.CodespaceType
	recentPrices               map[string]*utils.FixedSizeRing // symbol -> latest "numPricesStored" prices per "pricesStoreEvery" blocks
	am                         auth.AccountKeeper
	FeeManager                 *FeeManager
	RoundOrderFees             FeeHolder // order (and trade) related fee of this round, str of addr bytes -> fee
	CollectOrderInfoForPublish bool
	engines                    map[string]*me.MatchEng
	logger                     tmlog.Logger
	poolSize                   uint // number of concurrent channels, counted in the pow of 2
	cdc                        *wire.Codec
	OrderKeepers               []IDexOrderKeeper
}

var _ IDexKeeper = &DexKeeper{}

func NewDexKeeper(key sdk.StoreKey, tradingPairMapper store.TradingPairMapper, codespace sdk.CodespaceType, cdc *wire.Codec, am auth.AccountKeeper, collectOrderInfoForPublish bool, concurrency uint) *DexKeeper {
	logger := bnclog.With("module", "dex_keeper")
	return &DexKeeper{
		PairMapper:                 tradingPairMapper,
		storeKey:                   key,
		codespace:                  codespace,
		recentPrices:               make(map[string]*utils.FixedSizeRing, 256),
		am:                         am,
		RoundOrderFees:             make(map[string]*types.Fee, 256),
		FeeManager:                 NewFeeManager(cdc, logger),
		CollectOrderInfoForPublish: collectOrderInfoForPublish,
		engines:                    make(map[string]*me.MatchEng),
		poolSize:                   concurrency,
		cdc:                        cdc,
		logger:                     logger,
		OrderKeepers:               []IDexOrderKeeper{NewBEP2OrderKeeper(), NewMiniOrderKeeper()},
	}
}

func (kp *DexKeeper) InitRecentPrices(ctx sdk.Context) {
	kp.recentPrices = kp.PairMapper.GetRecentPrices(ctx, pricesStoreEvery, numPricesStored)
}

func (kp *DexKeeper) UpdateTickSizeAndLotSize(ctx sdk.Context) {
	tradingPairs := kp.PairMapper.ListAllTradingPairs(ctx)
	lotSizeCache := make(map[string]int64) // baseAsset -> lotSize
	for _, pair := range tradingPairs {
		if prices, ok := kp.recentPrices[pair.GetSymbol()]; ok && prices.Count() >= minimalNumPrices {
			priceWMA := dexUtils.CalcPriceWMA(prices)
			tickSize, lotSize := kp.determineTickAndLotSize(pair, priceWMA, lotSizeCache)
			if tickSize != pair.TickSize.ToInt64() ||
				lotSize != pair.LotSize.ToInt64() {
				ctx.Logger().Info("Updating tick/lotsize",
					"pair", pair.GetSymbol(), "old_ticksize", pair.TickSize, "new_ticksize", tickSize,
					"old_lotsize", pair.LotSize, "new_lotsize", lotSize)
				pair.TickSize = utils.Fixed8(tickSize)
				pair.LotSize = utils.Fixed8(lotSize)
				kp.PairMapper.AddTradingPair(ctx, pair)
			}
			kp.UpdateLotSize(pair.GetSymbol(), lotSize)
		} else {
			// keep the current tick_size/lot_size
			continue
		}
	}
}

func (kp *DexKeeper) determineTickAndLotSize(pair dexTypes.TradingPair, priceWMA int64, lotSizeCache map[string]int64) (tickSize, lotSize int64) {
	tickSize = dexUtils.CalcTickSize(priceWMA)
	if !sdk.IsUpgrade(upgrade.LotSizeOptimization) {
		lotSize = dexUtils.CalcLotSize(priceWMA)
		return
	}
	if lotSize, cached := lotSizeCache[pair.BaseAssetSymbol]; cached {
		return tickSize, lotSize
	}

	lotSize = kp.DetermineLotSize(pair.BaseAssetSymbol, pair.QuoteAssetSymbol, priceWMA)
	lotSizeCache[pair.BaseAssetSymbol] = lotSize
	return
}

func (kp *DexKeeper) DetermineLotSize(baseAssetSymbol, quoteAssetSymbol string, price int64) (lotSize int64) {
	var priceAgainstNative int64
	if baseAssetSymbol == types.NativeTokenSymbol {
		// price of BNB/BNB is 1e8
		priceAgainstNative = 1e8
	} else if quoteAssetSymbol == types.NativeTokenSymbol {
		priceAgainstNative = price
	} else {
		if ps, ok := kp.recentPrices[dexUtils.Assets2TradingPair(baseAssetSymbol, types.NativeTokenSymbol)]; ok {
			priceAgainstNative = dexUtils.CalcPriceWMA(ps)
		} else if ps, ok = kp.recentPrices[dexUtils.Assets2TradingPair(types.NativeTokenSymbol, baseAssetSymbol)]; ok {
			wma := dexUtils.CalcPriceWMA(ps)
			priceAgainstNative = 1e16 / wma
		} else {
			// the recentPrices still have not collected any price yet, iff the native pair is listed for less than kp.pricesStoreEvery blocks
			if engine, ok := kp.engines[dexUtils.Assets2TradingPair(baseAssetSymbol, types.NativeTokenSymbol)]; ok {
				priceAgainstNative = engine.LastTradePrice
			} else if engine, ok = kp.engines[dexUtils.Assets2TradingPair(types.NativeTokenSymbol, baseAssetSymbol)]; ok {
				priceAgainstNative = 1e16 / engine.LastTradePrice
			} else {
				// should not happen
				kp.logger.Error("DetermineLotSize failed because no native pair found", "base", baseAssetSymbol, "quote", quoteAssetSymbol)
			}
		}
	}
	lotSize = dexUtils.CalcLotSize(priceAgainstNative)
	return lotSize
}

func (kp *DexKeeper) UpdateLotSize(symbol string, lotSize int64) {
	eng, ok := kp.engines[symbol]
	if !ok {
		panic(fmt.Sprintf("match engine of symbol %s doesn't exist", symbol))
	}
	eng.LotSize = lotSize
}

func (kp *DexKeeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	symbol := strings.ToUpper(pair.GetSymbol())
	eng := CreateMatchEng(symbol, pair.ListPrice.ToInt64(), pair.LotSize.ToInt64())
	kp.engines[symbol] = eng
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			orderKeeper.initOrders(symbol)
			break
		}
	}
	return eng
}

func (kp *DexKeeper) AddOrder(info OrderInfo, isRecovery bool) (err error) {
	//try update order book first
	symbol := strings.ToUpper(info.Symbol)
	eng, ok := kp.engines[symbol]
	if !ok {
		err = fmt.Errorf("match engine of symbol %s doesn't exist", symbol)
		return
	}

	_, err = eng.Book.InsertOrder(info.Id, info.Side, info.CreatedHeight, info.Price, info.Quantity)
	if err != nil {
		return err
	}

	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			orderKeeper.addOrder(symbol, info, kp.CollectOrderInfoForPublish, isRecovery)
			break
		}
	}

	kp.logger.Debug("Added orders", "symbol", symbol, "id", info.Id)
	return nil
}

func orderNotFound(symbol, id string) error {
	return fmt.Errorf("Failed to find order [%v] on symbol [%v]", id, symbol)
}

func (kp *DexKeeper) RemoveOrder(id string, symbol string, postCancelHandler func(ord me.OrderPart)) (err error) {
	symbol = strings.ToUpper(symbol)
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			return orderKeeper.removeOrder(kp, id, symbol, postCancelHandler)
		}
	}
	return orderNotFound(symbol, id)
}

func (kp *DexKeeper) GetOrder(id string, symbol string, side int8, price int64) (ord me.OrderPart, err error) {
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

func (kp *DexKeeper) OrderExists(symbol, id string) (OrderInfo, bool) {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			return orderKeeper.orderExists(symbol, id)
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

func (kp *DexKeeper) SubscribeParamChange(hub *paramhub.Keeper) {
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
					panic("Load with no dex fee config ")
				}
			default:
				kp.logger.Debug("Receive param load that not interested.")
			}
		})
}

func (kp *DexKeeper) GetOrderBookLevels(pair string, maxLevels int) []store.OrderBookLevel {
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

func (kp *DexKeeper) GetOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(pair) {
			return orderKeeper.getOpenOrders(pair, addr)
		}
	}
	return make([]store.OpenOrder, 0)
}

func (kp *DexKeeper) GetOrderBooks(maxLevels int, pairType SymbolPairType) ChangedPriceLevelsMap {
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

func (kp *DexKeeper) GetPriceLevel(pair string, side int8, price int64) *me.PriceLevel {
	if eng, ok := kp.engines[pair]; ok {
		return eng.Book.GetPriceLevel(price, side)
	} else {
		return nil
	}
}

func (kp *DexKeeper) GetLastTrades(height int64, pair string) ([]me.Trade, int64) {
	if eng, ok := kp.engines[pair]; ok {
		if eng.LastMatchHeight == height {
			return eng.Trades, eng.LastTradePrice
		}
	}
	return nil, 0
}

// !!! FOR TEST USE ONLY
func (kp *DexKeeper) GetLastTradesForPair(pair string) ([]me.Trade, int64) {
	if eng, ok := kp.engines[pair]; ok {
		return eng.Trades, eng.LastTradePrice
	}
	return nil, 0
}

func (kp *DexKeeper) ClearOrderBook(pair string) {
	if eng, ok := kp.engines[pair]; ok {
		eng.Book.Clear()
	}
}

func (kp *DexKeeper) ClearOrderChanges() {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			orderKeeper.clearOrderChanges()
		}
	}
}

func (kp *DexKeeper) doTransfer(ctx sdk.Context, tran *Transfer) sdk.Error {
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

func (kp *DexKeeper) ClearAfterMatch() {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			orderKeeper.clearAfterMatch()
		}
	}
}

func (kp *DexKeeper) StoreTradePrices(ctx sdk.Context) {
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

func (kp *DexKeeper) allocate(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer)) (
	types.Fee, map[string]*types.Fee) {
	if !sdk.IsUpgrade(upgrade.BEP19) {
		return kp.allocateBeforeGalileo(ctx, tranCh, postAllocateHandler, kp.engines)
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
func (kp *DexKeeper) allocateBeforeGalileo(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer), engines map[string]*matcheng.MatchEng) (
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
		fee := kp.FeeManager.CalcTradeFee(acc.GetCoins(), in, engines)
		acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
		return fee
	})
	collectFee(expireInAsset, func(acc sdk.Account, in sdk.Coin) types.Fee {
		var i int64 = 0
		var fees types.Fee
		for ; i < in.Amount; i++ {
			fee := kp.FeeManager.CalcFixedFee(acc.GetCoins(), expireEventType, in.Denom, engines)
			acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
			fees.AddFee(fee)
		}
		return fees
	})
	return totalFee, feesPerAcc
}

func (kp *DexKeeper) allocateAndCalcFee(
	ctx sdk.Context,
	tradeOuts []chan Transfer,
	postAlloTransHandler TransferHandler) types.Fee {
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

func (kp *DexKeeper) expireOrders(ctx sdk.Context, blockTime time.Time) []chan Transfer {
	allOrders := make(map[string]map[string]*OrderInfo) //TODO replace by iterator
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			allOrders = appendAllOrdersMap(allOrders, orderKeeper.getAllOrders())
		}
	}
	size := len(allOrders)
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
			for symbol := range allOrders {
				symbolCh <- symbol
			}
			close(symbolCh)
		}, func() {
			for symbol := range symbolCh {
				engine := kp.engines[symbol]
				orders := allOrders[symbol]
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

func (kp *DexKeeper) ExpireOrders(
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

func (kp *DexKeeper) MarkBreatheBlock(ctx sdk.Context, height int64, blockTime time.Time) {
	key := utils.Int642Bytes(blockTime.Unix() / utils.SecondsPerDay)
	store := ctx.KVStore(kp.storeKey)
	bz, err := kp.cdc.MarshalBinaryBare(height)
	if err != nil {
		panic(err)
	}
	kp.logger.Debug(fmt.Sprintf("mark breathe block for key: %v (blockTime: %d), value: %v\n", key, blockTime.Unix(), bz))
	store.Set([]byte(key), bz)
}

func (kp *DexKeeper) GetBreatheBlockHeight(ctx sdk.Context, timeNow time.Time, daysBack int) (int64, error) {
	store := ctx.KVStore(kp.storeKey)
	t := timeNow.AddDate(0, 0, -daysBack).Unix()
	day := t / utils.SecondsPerDay
	bz := store.Get(utils.Int642Bytes(day))
	if bz == nil {
		return 0, fmt.Errorf("breathe block not found for day %v", day)
	}

	var height int64
	err := kp.cdc.UnmarshalBinaryBare(bz, &height)
	if err != nil {
		panic(err)
	}
	return height, nil
}

func (kp *DexKeeper) GetLastBreatheBlockHeight(ctx sdk.Context, latestBlockHeight int64, timeNow time.Time, blockInterval, daysBack int) int64 {
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
func (kp *DexKeeper) updateRoundOrderFee(addr string, fee types.Fee) {
	if existingFee, ok := kp.RoundOrderFees[addr]; ok {
		existingFee.AddFee(fee)
	} else {
		kp.RoundOrderFees[addr] = &fee
	}
}

func (kp *DexKeeper) ClearRoundFee() {
	kp.RoundOrderFees = make(map[string]*types.Fee, 256)
}

func (kp *DexKeeper) CanDelistTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error {
	// trading pair against native token should not be delisted if there is any other trading pair exist
	baseAsset = strings.ToUpper(baseAsset)
	quoteAsset = strings.ToUpper(quoteAsset)

	if baseAsset == quoteAsset {
		return fmt.Errorf("base asset symbol should not be identical to quote asset symbol")
	}

	if !kp.PairMapper.Exists(ctx, baseAsset, quoteAsset) {
		return fmt.Errorf("trading pair %s_%s does not exist", baseAsset, quoteAsset)
	}

	if baseAsset != types.NativeTokenSymbol && quoteAsset != types.NativeTokenSymbol {
		return nil
	}

	var symbolToCheck string
	if baseAsset != types.NativeTokenSymbol {
		symbolToCheck = baseAsset
	} else {
		symbolToCheck = quoteAsset
	}

	tradingPairs := kp.PairMapper.ListAllTradingPairs(ctx)
	for _, pair := range tradingPairs { //TODO
		if (pair.BaseAssetSymbol == symbolToCheck && pair.QuoteAssetSymbol != types.NativeTokenSymbol) ||
			(pair.QuoteAssetSymbol == symbolToCheck && pair.BaseAssetSymbol != types.NativeTokenSymbol) {
			return fmt.Errorf("trading pair %s_%s should not exist before delisting %s_%s",
				pair.BaseAssetSymbol, pair.QuoteAssetSymbol, baseAsset, quoteAsset)
		}
	}

	return nil
}

func (kp *DexKeeper) DelistTradingPair(ctx sdk.Context, symbol string, postAllocTransHandler TransferHandler) {
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
	delete(kp.recentPrices, symbol)

	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			orderKeeper.deleteOrdersForPair(symbol)
			break
		}
	}

	baseAsset, quoteAsset := dexUtils.TradingPair2AssetsSafe(symbol)
	err := kp.PairMapper.DeleteTradingPair(ctx, baseAsset, quoteAsset)
	if err != nil {
		kp.logger.Error("delete trading pair error", "err", err.Error())
	}
}

func (kp *DexKeeper) expireAllOrders(ctx sdk.Context, symbol string) []chan Transfer {
	ordersOfSymbol := make(map[string]*OrderInfo)
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			ordersOfSymbol = orderKeeper.getAllOrdersForPair(symbol)
			break
		}
	}

	orderNum := len(ordersOfSymbol)
	if orderNum == 0 {
		kp.logger.Info("no orders to expire", "symbol", symbol)
		return nil
	}

	concurrency := 1 << kp.poolSize
	channelSize := orderNum / concurrency

	transferChs := make([]chan Transfer, concurrency)
	for i := range transferChs {
		transferChs[i] = make(chan Transfer, channelSize)
	}

	expire := func(orders map[string]*OrderInfo, engine *me.MatchEng, side int8) {
		_ = engine.Book.RemoveOrders(math.MaxInt64, side, func(ord me.OrderPart) {
			// gen transfer
			if ordMsg, ok := orders[ord.Id]; ok && ordMsg != nil {
				h := channelHash(ordMsg.Sender, concurrency)
				transferChs[h] <- TransferFromExpired(ord, *ordMsg)
			} else {
				kp.logger.Error("failed to locate order to remove in order book", "oid", ord.Id)
			}
		})
	}

	go func() {
		engine := kp.engines[symbol]
		orders := ordersOfSymbol
		expire(orders, engine, me.BUYSIDE)
		expire(orders, engine, me.SELLSIDE)

		for _, transferCh := range transferChs {
			close(transferCh)
		}
	}()

	return transferChs
}

func (kp *DexKeeper) CanListTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error {
	// trading pair against native token should exist if quote token is not native token
	baseAsset = strings.ToUpper(baseAsset)
	quoteAsset = strings.ToUpper(quoteAsset)

	if baseAsset == quoteAsset {
		return fmt.Errorf("base asset symbol should not be identical to quote asset symbol")
	}

	if kp.PairMapper.Exists(ctx, baseAsset, quoteAsset) || kp.PairMapper.Exists(ctx, quoteAsset, baseAsset) {
		return errors.New("trading pair exists")
	}

	if baseAsset != types.NativeTokenSymbol &&
		quoteAsset != types.NativeTokenSymbol {

		if !kp.PairMapper.Exists(ctx, baseAsset, types.NativeTokenSymbol) &&
			!kp.PairMapper.Exists(ctx, types.NativeTokenSymbol, baseAsset) {
			return fmt.Errorf("token %s should be listed against BNB before against %s",
				baseAsset, quoteAsset)
		}

		if !kp.PairMapper.Exists(ctx, quoteAsset, types.NativeTokenSymbol) &&
			!kp.PairMapper.Exists(ctx, types.NativeTokenSymbol, quoteAsset) {
			return fmt.Errorf("token %s should be listed against BNB before listing %s against %s",
				quoteAsset, baseAsset, quoteAsset)
		}
	}

	if isMiniSymbolPair(baseAsset, quoteAsset) && types.NativeTokenSymbol != quoteAsset { //todo permit BUSD
		return errors.New("quote token is not valid for mini symbol pair: " + quoteAsset)
	}

	return nil
}

func (kp *DexKeeper) GetAllOrders() map[string]map[string]*OrderInfo {
	allOrders := make(map[string]map[string]*OrderInfo) //TODO replace by iterator
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			allOrders = appendAllOrdersMap(allOrders, orderKeeper.getAllOrders())
		}
	}
	return allOrders
}

func (kp *DexKeeper) GetAllOrdersForPair(symbol string) map[string]*OrderInfo {
	ordersOfSymbol := make(map[string]*OrderInfo)
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			ordersOfSymbol = orderKeeper.getAllOrdersForPair(symbol)
			break
		}
	}
	return ordersOfSymbol
}

func (kp *DexKeeper) ReloadOrder(symbol string, orderInfo *OrderInfo, height int64) {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			orderKeeper.reloadOrder(symbol, orderInfo, height, kp.CollectOrderInfoForPublish)
			return
		}
	}
}

func (kp *DexKeeper) ValidateOrder(context sdk.Context, account sdk.Account, msg NewOrderMsg) error {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(msg.Symbol) {
			return orderKeeper.validateOrder(kp, context, account, msg)
		}
	}
	return fmt.Errorf("symbol:%s is not supported", msg.Symbol)
}


func (kp *DexKeeper) GetOrderChanges(pairType SymbolPairType) OrderChanges {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportPairType(pairType) {
			return orderKeeper.getOrderChanges()
		}
	}
	kp.logger.Error("pairType is not supported %d", pairType)
	return make(OrderChanges, 0)
}

func (kp *DexKeeper) UpdateOrderChange(change OrderChange, symbol string) {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			orderKeeper.appendOrderChange(change)
			return
		}
	}
	kp.logger.Error("symbol is not supported %d", symbol)
}

func (kp *DexKeeper) UpdateOrderChangeSync(change OrderChange, symbol string) {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.support(symbol) {
			orderKeeper.appendOrderChangeSync(change)
			return
		}
	}
	kp.logger.Error("symbol is not supported %d", symbol)
}

func (kp *DexKeeper) GetOrderInfosForPub(pairType SymbolPairType) OrderInfoForPublish {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportPairType(pairType) {
			return orderKeeper.getOrderInfosForPub()
		}
	}
	kp.logger.Error("pairType is not supported %d", pairType)
	return make(OrderInfoForPublish)

}

func (kp *DexKeeper) GetPairMapper() store.TradingPairMapper {
	return kp.PairMapper
}

func (kp *DexKeeper) getAccountKeeper() *auth.AccountKeeper {
	return &kp.am
}

func (kp *DexKeeper) getLogger() tmlog.Logger {
	return kp.logger
}

func (kp *DexKeeper) getFeeManager() *FeeManager {
	return kp.FeeManager
}

func (kp *DexKeeper) ShouldPublishOrder() bool {
	return kp.CollectOrderInfoForPublish
}

func (kp *DexKeeper) GetEngines() map[string]*me.MatchEng {
	return kp.engines
}
func appendAllOrdersMap(ms ...map[string]map[string]*OrderInfo) map[string]map[string]*OrderInfo {
	res := make(map[string]map[string]*OrderInfo)
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

func CreateMatchEng(pairSymbol string, basePrice, lotSize int64) *me.MatchEng {
	return me.NewMatchEng(pairSymbol, basePrice, lotSize, 0.05)
}

func isMiniSymbolPair(baseAsset, quoteAsset string) bool {
	if sdk.IsUpgradeHeight(upgrade.BEP8) {
		return types.IsMiniTokenSymbol(baseAsset) || types.IsMiniTokenSymbol(quoteAsset)
	}
	return false
}
