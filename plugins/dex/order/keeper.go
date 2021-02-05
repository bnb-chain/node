package order

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	dbm "github.com/tendermint/tendermint/libs/db"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmstore "github.com/tendermint/tendermint/store"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/auth"
	paramhub "github.com/cosmos/cosmos-sdk/x/paramHub/keeper"
	paramTypes "github.com/cosmos/cosmos-sdk/x/paramHub/types"

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
	preferencePriceLevel = 500
)

var BUSDSymbol string

type FeeHandler func(map[string]*sdk.Fee)
type TransferHandler func(Transfer)

type DexKeeper struct {
	PairMapper                 store.TradingPairMapper
	storeKey                   sdk.StoreKey // The key used to access the store from the Context.
	codespace                  sdk.CodespaceType
	recentPrices               map[string]*utils.FixedSizeRing // symbol -> latest "numPricesStored" prices per "pricesStoreEvery" blocks
	am                         auth.AccountKeeper
	FeeManager                 *FeeManager
	RoundOrderFees             FeeHolder // order (and trade) related fee of this round, str of addr bytes -> fee
	CollectOrderInfoForPublish bool      //TODO separate for each order keeper
	engines                    map[string]*me.MatchEng
	pairsType                  map[string]dexTypes.SymbolPairType
	logger                     tmlog.Logger
	poolSize                   uint // number of concurrent channels, counted in the pow of 2
	cdc                        *wire.Codec
	OrderKeepers               []DexOrderKeeper
}

func NewDexKeeper(key sdk.StoreKey, am auth.AccountKeeper, tradingPairMapper store.TradingPairMapper, codespace sdk.CodespaceType, concurrency uint, cdc *wire.Codec, collectOrderInfoForPublish bool) *DexKeeper {
	logger := bnclog.With("module", "dexkeeper")
	mainMarketOrderKeeper, growthMarketOrderKeeper := NewMainMarketOrderKeeper(), NewGrowthMarketOrderKeeper()
	if collectOrderInfoForPublish {
		mainMarketOrderKeeper.enablePublish()
		growthMarketOrderKeeper.enablePublish()
	}

	return &DexKeeper{
		PairMapper:                 tradingPairMapper,
		storeKey:                   key,
		codespace:                  codespace,
		recentPrices:               make(map[string]*utils.FixedSizeRing, 256),
		am:                         am,
		RoundOrderFees:             make(map[string]*sdk.Fee, 256),
		FeeManager:                 NewFeeManager(cdc, logger),
		CollectOrderInfoForPublish: collectOrderInfoForPublish,
		engines:                    make(map[string]*me.MatchEng),
		pairsType:                  make(map[string]dexTypes.SymbolPairType),
		poolSize:                   concurrency,
		cdc:                        cdc,
		logger:                     logger,
		OrderKeepers:               []DexOrderKeeper{mainMarketOrderKeeper, growthMarketOrderKeeper},
	}
}

func (kp *DexKeeper) Init(ctx sdk.Context, blockInterval, daysBack int, blockStore *tmstore.BlockStore, stateDB dbm.DB, lastHeight int64, txDecoder sdk.TxDecoder) {
	kp.initOrderBook(ctx, blockInterval, daysBack, blockStore, stateDB, lastHeight, txDecoder)
	kp.InitRecentPrices(ctx)
}

func (kp *DexKeeper) InitRecentPrices(ctx sdk.Context) {
	kp.recentPrices = kp.PairMapper.GetRecentPrices(ctx, pricesStoreEvery, numPricesStored)
}

func (kp *DexKeeper) SetBUSDSymbol(symbol string) {
	BUSDSymbol = symbol
}

func (kp *DexKeeper) EnablePublish() {
	kp.CollectOrderInfoForPublish = true
	for i := range kp.OrderKeepers {
		kp.OrderKeepers[i].enablePublish()
	}
}

func (kp *DexKeeper) GetPairType(symbol string) dexTypes.SymbolPairType {
	pairType, ok := kp.pairsType[symbol]
	if !ok {
		err := fmt.Errorf("unknown type of symbol: %s", symbol)
		kp.logger.Error(err.Error())
		return dexTypes.PairType.UNKNOWN
	}
	return pairType
}

func (kp *DexKeeper) getOrderKeeper(symbol string) (DexOrderKeeper, error) {
	pairType, ok := kp.pairsType[symbol]
	if !ok {
		err := fmt.Errorf("invalid symbol: %s", symbol)
		kp.logger.Debug(err.Error())
		return nil, err
	}
	for i := range kp.OrderKeepers {
		if kp.OrderKeepers[i].supportPairType(pairType) {
			return kp.OrderKeepers[i], nil
		}
	}
	err := fmt.Errorf("failed to find orderKeeper for symbol pair [%s]", symbol)
	kp.logger.Error(err.Error())
	return nil, err
}

func (kp *DexKeeper) mustGetOrderKeeper(symbol string) DexOrderKeeper {
	pairType := kp.pairsType[symbol]
	for i := range kp.OrderKeepers {
		if kp.OrderKeepers[i].supportPairType(pairType) {
			return kp.OrderKeepers[i]
		}
	}

	panic(fmt.Errorf("invalid symbol %s", symbol))
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
		var found bool
		priceAgainstNative, found = kp.calcPriceAgainst(baseAssetSymbol, types.NativeTokenSymbol)
		if !found {
			if sdk.IsUpgrade(upgrade.BEP70) && len(BUSDSymbol) > 0 {
				var tmp = big.NewInt(0)
				priceAgainstBUSD, ok := kp.calcPriceAgainst(baseAssetSymbol, BUSDSymbol)
				if !ok {
					// for newly added pair, there is no trading yet
					priceAgainstBUSD = price
				}
				priceBUSDAgainstNative, _ := kp.calcPriceAgainst(BUSDSymbol, types.NativeTokenSymbol)
				tmp = tmp.Div(tmp.Mul(big.NewInt(priceAgainstBUSD), big.NewInt(priceBUSDAgainstNative)), big.NewInt(1e8))
				if tmp.IsInt64() {
					priceAgainstNative = tmp.Int64()
				} else {
					priceAgainstNative = math.MaxInt64
				}
			} else {
				// should not happen
				kp.logger.Error("DetermineLotSize failed because no native pair found", "base", baseAssetSymbol, "quote", quoteAssetSymbol)
			}
		}
	}
	lotSize = dexUtils.CalcLotSize(priceAgainstNative)
	return lotSize
}

func (kp *DexKeeper) calcPriceAgainst(symbol, targetSymbol string) (int64, bool) {
	var priceAgainst int64 = 0
	var found bool
	if ps, ok := kp.recentPrices[dexUtils.Assets2TradingPair(symbol, targetSymbol)]; ok {
		priceAgainst = dexUtils.CalcPriceWMA(ps)
		found = true
	} else if ps, ok = kp.recentPrices[dexUtils.Assets2TradingPair(targetSymbol, symbol)]; ok {
		wma := dexUtils.CalcPriceWMA(ps)
		priceAgainst = 1e16 / wma
		found = true
	} else {
		// the recentPrices still have not collected any price yet, iff the native pair is listed for less than kp.pricesStoreEvery blocks
		if engine, ok := kp.engines[dexUtils.Assets2TradingPair(symbol, targetSymbol)]; ok {
			priceAgainst = engine.LastTradePrice
			found = true
		} else if engine, ok = kp.engines[dexUtils.Assets2TradingPair(targetSymbol, symbol)]; ok {
			priceAgainst = 1e16 / engine.LastTradePrice
			found = true
		}
	}

	return priceAgainst, found
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
	var pairType dexTypes.SymbolPairType
	if sdk.IsUpgrade(upgrade.BEPX) && !sdk.IsUpgradeHeight(upgrade.BEPX) {
		pairType = pair.PairType
	} else {
		pairType = dexTypes.PairType.BEP2
		if dexUtils.IsMiniTokenTradingPair(symbol) {
			pairType = dexTypes.PairType.MINI
		}
	}
	kp.pairsType[symbol] = pairType
	for i := range kp.OrderKeepers {
		if kp.OrderKeepers[i].supportPairType(pairType) {
			kp.OrderKeepers[i].initOrders(symbol)
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

	kp.mustGetOrderKeeper(symbol).addOrder(symbol, info, isRecovery)
	kp.logger.Debug("Added orders", "symbol", symbol, "id", info.Id)
	return nil
}

func orderNotFound(symbol, id string) error {
	return fmt.Errorf("Failed to find order [%v] on symbol [%v]", id, symbol)
}

func (kp *DexKeeper) RemoveOrder(id string, symbol string, postCancelHandler func(ord me.OrderPart)) error {
	symbol = strings.ToUpper(symbol)
	if dexOrderKeeper, err := kp.getOrderKeeper(symbol); err == nil {
		ord, err := dexOrderKeeper.removeOrder(kp, id, symbol)
		if err != nil {
			return err
		}
		if postCancelHandler != nil {
			postCancelHandler(ord)
		}
		return nil
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
	if dexOrderKeeper, err := kp.getOrderKeeper(symbol); err == nil {
		return dexOrderKeeper.orderExists(symbol, id)
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
		func(_ sdk.Context, iChange interface{}) {
			switch change := iChange.(type) {
			case []paramTypes.FeeParam:
				feeConfig := ParamToFeeConfig(change)
				if feeConfig != nil {
					kp.FeeManager.UpdateConfig(*feeConfig)
				}
			default:
				kp.logger.Debug("Receive param changes that not interested.")
			}
		},
		nil,
		func(context sdk.Context, iState interface{}) {
			switch state := iState.(type) {
			case paramTypes.GenesisState:
				feeConfig := ParamToFeeConfig(state.FeeGenesis)
				if feeConfig != nil {
					kp.FeeManager.UpdateConfig(*feeConfig)
				} else {
					panic("Genesis with no dex fee config ")
				}
			default:
				kp.logger.Debug("Receive param genesis state that not interested.")
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

func (kp *DexKeeper) GetOrderBookLevels(pair string, maxLevels int) (orderbook []store.OrderBookLevel, pendingMatch bool) {
	orderbook = make([]store.OrderBookLevel, maxLevels)

	i, j := 0, 0
	if eng, ok := kp.engines[pair]; ok {
		// TODO: check considered bucket splitting?
		eng.Book.ShowDepth(maxLevels, func(p *me.PriceLevel, levelIndex int) {
			orderbook[i].BuyPrice = utils.Fixed8(p.Price)
			orderbook[i].BuyQty = utils.Fixed8(p.TotalLeavesQty())
			i++
		}, func(p *me.PriceLevel, levelIndex int) {
			orderbook[j].SellPrice = utils.Fixed8(p.Price)
			orderbook[j].SellQty = utils.Fixed8(p.TotalLeavesQty())
			j++
		})
		roundOrders := kp.mustGetOrderKeeper(pair).getRoundOrdersForPair(pair)
		pendingMatch = len(roundOrders) > 0
	}
	return orderbook, pendingMatch
}

func (kp *DexKeeper) GetOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder {
	if dexOrderKeeper, err := kp.getOrderKeeper(pair); err == nil {
		return dexOrderKeeper.getOpenOrders(pair, addr)
	}
	return make([]store.OpenOrder, 0)
}

func (kp *DexKeeper) GetOrderBooks(maxLevels int) ChangedPriceLevelsMap {
	var res = make(ChangedPriceLevelsMap)
	for pair, eng := range kp.engines {
		buys := make(map[int64]int64)
		sells := make(map[int64]int64)
		res[pair] = ChangedPriceLevelsPerSymbol{buys, sells}

		// TODO: check considered bucket splitting?
		eng.Book.ShowDepth(maxLevels, func(p *me.PriceLevel, levelIndex int) {
			buys[p.Price] = p.TotalLeavesQty()
		}, func(p *me.PriceLevel, levelIndex int) {
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

func (kp *DexKeeper) MigrateTradingPairType(ctx sdk.Context) {
	kp.MigrateKeeperTradingPairType()
	kp.PairMapper.MigrateToMainAndGrowthMarket(ctx)
}

func (kp *DexKeeper) MigrateKeeperTradingPairType() {

	for pairSymbol, pairType := range kp.pairsType {
		kp.logger.Debug("Migrate pair %s, type %v", pairSymbol, pairType)
		if pairType == dexTypes.PairType.BEP2 {
			kp.pairsType[pairSymbol] = dexTypes.PairType.MAIN
		} else if pairType == dexTypes.PairType.MINI {
			kp.pairsType[pairSymbol] = dexTypes.PairType.GROWTH
		}
	}

}

func (kp *DexKeeper) PromoteGrowthPairToMainMarket(ctx sdk.Context, pairs []string) {
	for _, pair := range pairs {
		kp.PromoteGrowthToMain(pair)
		kp.PairMapper.PromoteGrowthToMain(ctx, pair)
	}
}

func (kp *DexKeeper) PromoteGrowthToMain(pair string) {
	pairType, ok := kp.pairsType[pair]
	if !ok {
		kp.logger.Error("failed to find pairType of symbol", "symbol", pair)
		return
	}
	if pairType != dexTypes.PairType.GROWTH {
		kp.logger.Error("pairType of symbol is not correct", "symbol", pair, "pairType", pairType)
		return
	}
	orders := kp.mustGetOrderKeeper(pair).getAllOrdersForPair(pair)
	kp.mustGetOrderKeeper(pair).deleteOrdersForPair(pair)

	kp.pairsType[pair] = dexTypes.PairType.MAIN

	for _, orderInfo := range orders {
		kp.mustGetOrderKeeper(pair).addToAllOrders(pair, *orderInfo)
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
	sdk.Fee, map[string]*sdk.Fee) {
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
	var totalFee sdk.Fee
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

	feesPerAcc := make(map[string]*sdk.Fee)
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
func (kp *DexKeeper) allocateBeforeGalileo(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer)) (
	sdk.Fee, map[string]*sdk.Fee) {
	// use string of the addr as the key since map makes a fast path for string key.
	// Also, making the key have same length is also an optimization.
	tradeInAsset := make(map[string]*sortedAsset)
	// expire fee is fixed, so we count by numbers.
	expireInAsset := make(map[string]*sortedAsset)
	// we need to distinguish different expire event, IOCExpire or Expire. only one of the two will exist.
	var expireEventType transferEventType
	var totalFee sdk.Fee
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

	feesPerAcc := make(map[string]*sdk.Fee)
	collectFee := func(assetsMap map[string]*sortedAsset, calcFeeAndDeduct func(acc sdk.Account, in sdk.Coin) sdk.Fee) {
		for addrStr, assets := range assetsMap {
			addr := sdk.AccAddress(addrStr)
			acc := kp.am.GetAccount(ctx, addr)

			var fees sdk.Fee
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
	collectFee(tradeInAsset, func(acc sdk.Account, in sdk.Coin) sdk.Fee {
		fee := kp.FeeManager.CalcTradeFee(acc.GetCoins(), in, kp.engines)
		acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
		return fee
	})
	collectFee(expireInAsset, func(acc sdk.Account, in sdk.Coin) sdk.Fee {
		var i int64 = 0
		var fees sdk.Fee
		for ; i < in.Amount; i++ {
			fee := kp.FeeManager.CalcFixedFee(acc.GetCoins(), expireEventType, in.Denom, kp.engines)
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
	postAlloTransHandler TransferHandler) sdk.Fee {
	concurrency := len(tradeOuts)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	feesPerCh := make([]sdk.Fee, concurrency)
	feesPerAcc := make([]map[string]*sdk.Fee, concurrency)
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
	totalFee := sdk.Fee{}
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

	expireHeight, forceExpireHeight, err := kp.getExpireHeight(ctx, blockTime)
	if err != nil {
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
		removeCallback := func(ord me.OrderPart) {
			// gen transfer
			if ordMsg, ok := orders[ord.Id]; ok && ordMsg != nil {
				h := channelHash(ordMsg.Sender, concurrency)
				transferChs[h] <- TransferFromExpired(ord, *ordMsg)
				// delete from allOrders
				delete(orders, ord.Id)
			} else {
				kp.logger.Error("failed to locate order to remove in order book", "oid", ord.Id)
			}
		}
		if !sdk.IsUpgrade(upgrade.BEP67) {
			engine.Book.RemoveOrders(expireHeight, side, removeCallback)
		} else {
			engine.Book.RemoveOrdersBasedOnPriceLevel(expireHeight, forceExpireHeight, preferencePriceLevel, side, removeCallback)
		}
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

func (kp *DexKeeper) getExpireHeight(ctx sdk.Context, blockTime time.Time) (expireHeight, forceExpireHeight int64, noBreatheBlock error) {
	const effectiveDays = 3
	expireHeight, noBreatheBlock = kp.GetBreatheBlockHeight(ctx, blockTime, effectiveDays)
	if noBreatheBlock != nil {
		// breathe block not found, that should only happens in in the first three days, just log it and ignore.
		kp.logger.Error(noBreatheBlock.Error())
		return -1, -1, noBreatheBlock
	}

	if sdk.IsUpgrade(upgrade.BEP67) {
		const forceExpireDays = 30
		var err error
		forceExpireHeight, err = kp.GetBreatheBlockHeight(ctx, blockTime, forceExpireDays)
		if err != nil {
			//if breathe block of 30 days ago not found, the breathe block of 3 days ago still can be processed, so return err=nil
			kp.logger.Error(err.Error())
			return expireHeight, -1, nil
		}
	} else {
		forceExpireHeight = -1
	}
	return expireHeight, forceExpireHeight, nil
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
func (kp *DexKeeper) updateRoundOrderFee(addr string, fee sdk.Fee) {
	if existingFee, ok := kp.RoundOrderFees[addr]; ok {
		existingFee.AddFee(fee)
	} else {
		kp.RoundOrderFees[addr] = &fee
	}
}

func (kp *DexKeeper) ClearRoundFee() {
	kp.RoundOrderFees = make(map[string]*sdk.Fee, 256)
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
	kp.deleteRecentPrices(ctx, symbol)
	kp.mustGetOrderKeeper(symbol).deleteOrdersForPair(symbol)

	baseAsset, quoteAsset := dexUtils.TradingPair2AssetsSafe(symbol)
	err := kp.PairMapper.DeleteTradingPair(ctx, baseAsset, quoteAsset)
	if err != nil {
		kp.logger.Error("delete trading pair error", "err", err.Error())
	}
}

func (kp *DexKeeper) deleteRecentPrices(ctx sdk.Context, symbol string) {
	delete(kp.recentPrices, symbol)
	kp.PairMapper.DeleteRecentPrices(ctx, symbol)
}

func (kp *DexKeeper) expireAllOrders(ctx sdk.Context, symbol string) []chan Transfer {
	ordersOfSymbol := make(map[string]*OrderInfo)
	if dexOrderKeeper, err := kp.getOrderKeeper(symbol); err == nil {
		ordersOfSymbol = dexOrderKeeper.getAllOrdersForPair(symbol)
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

	if kp.pairExistsBetween(ctx, baseAsset, quoteAsset) {
		return errors.New("trading pair exists")
	}

	if baseAsset != types.NativeTokenSymbol &&
		quoteAsset != types.NativeTokenSymbol {

		// support busd pair listing including mini-token as base
		if sdk.IsUpgrade(upgrade.BEP70) && len(BUSDSymbol) > 0 {
			if baseAsset == BUSDSymbol || quoteAsset == BUSDSymbol {
				if kp.pairExistsBetween(ctx, types.NativeTokenSymbol, BUSDSymbol) {
					return nil
				}
			}
		}

		if !kp.pairExistsBetween(ctx, types.NativeTokenSymbol, baseAsset) {
			return fmt.Errorf("token %s should be listed against BNB before against %s",
				baseAsset, quoteAsset)
		}

		if !kp.pairExistsBetween(ctx, types.NativeTokenSymbol, quoteAsset) {
			return fmt.Errorf("token %s should be listed against BNB before listing %s against %s",
				quoteAsset, baseAsset, quoteAsset)
		}
	}

	return nil
}

func (kp *DexKeeper) GetAllOrders() map[string]map[string]*OrderInfo {
	allOrders := make(map[string]map[string]*OrderInfo)
	for _, orderKeeper := range kp.OrderKeepers {
		allOrders = appendAllOrdersMap(allOrders, orderKeeper.getAllOrders())
	}
	return allOrders
}

// ONLY FOR TEST USE
func (kp *DexKeeper) GetAllOrdersForPair(symbol string) map[string]*OrderInfo {
	return kp.mustGetOrderKeeper(symbol).getAllOrdersForPair(symbol)
}

func (kp *DexKeeper) ReloadOrder(symbol string, orderInfo *OrderInfo, height int64) {
	kp.mustGetOrderKeeper(symbol).reloadOrder(symbol, orderInfo, height)
}

func (kp *DexKeeper) GetOrderChanges(pairType dexTypes.SymbolPairType) OrderChanges {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportPairType(pairType) {
			return orderKeeper.getOrderChanges()
		}
	}
	kp.logger.Error("pairType is not supported %d for OrderChanges", pairType)
	return make(OrderChanges, 0)
}
func (kp *DexKeeper) GetAllOrderChanges() OrderChanges {
	var res OrderChanges
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			res = append(res, orderKeeper.getOrderChanges()...)
		}
	}
	return res
}

func (kp *DexKeeper) UpdateOrderChangeSync(change OrderChange, symbol string) {
	if dexOrderKeeper, err := kp.getOrderKeeper(symbol); err == nil {
		dexOrderKeeper.appendOrderChangeSync(change)
		return
	}
	kp.logger.Error("symbol is not supported %d", symbol)
}

func (kp *DexKeeper) GetOrderInfosForPub(pairType dexTypes.SymbolPairType) OrderInfoForPublish {
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportPairType(pairType) {
			return orderKeeper.getOrderInfosForPub()
		}
	}
	kp.logger.Error("pairType is not supported %d for OrderInfosForPub", pairType)
	return make(OrderInfoForPublish)
}

func (kp *DexKeeper) GetAllOrderInfosForPub() OrderInfoForPublish {
	orderInfoForPub := make(OrderInfoForPublish)
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			orderInfoForPub = appendOrderInfoForPub(orderInfoForPub, orderKeeper.getOrderInfosForPub())
		}
	}
	return orderInfoForPub
}

func (kp *DexKeeper) RemoveOrderInfosForPub(pair string, orderId string) {
	if orderKeeper, err := kp.getOrderKeeper(pair); err == nil {
		orderKeeper.removeOrderInfosForPub(orderId)
		return
	}

	kp.logger.Error("pair is not supported %d", pair)
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

func appendOrderInfoForPub(ms ...OrderInfoForPublish) OrderInfoForPublish {
	res := make(OrderInfoForPublish)
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
	return types.IsMiniTokenSymbol(baseAsset) || types.IsMiniTokenSymbol(quoteAsset)
}

// Check whether there is trading pair between two symbols
func (kp *DexKeeper) pairExistsBetween(ctx sdk.Context, symbolA, symbolB string) bool {
	return kp.PairMapper.Exists(ctx, symbolA, symbolB) || kp.PairMapper.Exists(ctx, symbolB, symbolA)
}
