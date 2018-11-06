package pub

import (
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/types"
	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

func GetAccountBalances(mapper auth.AccountKeeper, ctx sdk.Context, accSlices ...[]string) (res map[string]Account) {
	res = make(map[string]Account)

	for _, accs := range accSlices {
		for _, addrBytesStr := range accs {
			if _, ok := res[addrBytesStr]; !ok {
				addr := sdk.AccAddress([]byte(addrBytesStr))
				if acc, ok := mapper.GetAccount(ctx, addr).(types.NamedAccount); ok {
					assetsMap := make(map[string]*AssetBalance)
					// TODO(#66): set the length to be the total coins this account owned
					assets := make([]*AssetBalance, 0, 10)

					for _, freeCoin := range acc.GetCoins() {
						if assetBalance, ok := assetsMap[freeCoin.Denom]; ok {
							assetBalance.Free = freeCoin.Amount.Int64()
						} else {
							newAB := &AssetBalance{Asset: freeCoin.Denom, Free: freeCoin.Amount.Int64()}
							assets = append(assets, newAB)
							assetsMap[freeCoin.Denom] = newAB
						}
					}

					for _, frozenCoin := range acc.GetFrozenCoins() {
						if assetBalance, ok := assetsMap[frozenCoin.Denom]; ok {
							assetBalance.Frozen = frozenCoin.Amount.Int64()
						} else {
							newAB := &AssetBalance{Asset: frozenCoin.Denom, Frozen: frozenCoin.Amount.Int64()}
							assets = append(assets, newAB)
							assetsMap[frozenCoin.Denom] = newAB
						}
					}

					for _, lockedCoin := range acc.GetLockedCoins() {
						if assetBalance, ok := assetsMap[lockedCoin.Denom]; ok {
							assetBalance.Locked = lockedCoin.Amount.Int64()
						} else {
							newAB := &AssetBalance{Asset: lockedCoin.Denom, Locked: lockedCoin.Amount.Int64()}
							assets = append(assets, newAB)
							assetsMap[lockedCoin.Denom] = newAB
						}
					}

					bech32Str := addr.String()
					res[bech32Str] = Account{bech32Str, assets}
				} else {
					Logger.Error(fmt.Sprintf("failed to get account %s from AccountKeeper", addr.String()))
				}
			}
		}
	}

	return
}

func MatchAndAllocateAllForPublish(
	dexKeeper *orderPkg.Keeper,
	ctx sdk.Context) []*Trade {
	// These two channels are used for protect not update `tradesToPublish` and `dexKeeper.OrderChanges` concurrently
	// matcher would send item to feeCollectorForTrades in several goroutine (well-designed)
	// while tradesToPublish and dexKeeper.OrderChanges are not separated by concurrent factor (users here), so we have
	// to organized transfer holders into 2 channels
	tradeHolderCh := make(chan orderPkg.TradeHolder, TransferCollectionChannelSize)
	iocExpireFeeHolderCh := make(chan orderPkg.ExpireHolder, TransferCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(2)

	tradesToPublish := make([]*Trade, 0)
	go collectTradeForPublish(&tradesToPublish, &wg, ctx.BlockHeader().Height, tradeHolderCh)
	go updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh, orderPkg.IocNoFill)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		if tran.IsExpire() {
			iocExpireFeeHolderCh <- orderPkg.ExpireHolder{tran.Oid}
		} else {
			tradeHolderCh <- orderPkg.TradeHolder{tran.Oid, tran.Trade, tran.Symbol}
		}
	}
	ctx = dexKeeper.MatchAndAllocateAll(ctx, feeCollectorForTrades, setFeeHolder)
	close(tradeHolderCh)
	close(iocExpireFeeHolderCh)
	wg.Wait()

	return tradesToPublish
}

func ExpireOrdersForPublish(
	dexKeeper *orderPkg.Keeper,
	ctx sdk.Context,
	blockTime time.Time) {
	expireHolderCh := make(chan orderPkg.ExpireHolder, TransferCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go updateExpireFeeForPublish(dexKeeper, &wg, expireHolderCh, orderPkg.Expired)
	var collectorForExpires = func(tran orderPkg.Transfer) {
		if tran.IsExpire() {
			expireHolderCh <- orderPkg.ExpireHolder{tran.Oid}
		}
	}
	dexKeeper.ExpireOrders(ctx, blockTime, collectorForExpires, setFeeHolder)
	close(expireHolderCh)
	wg.Wait()
}

// for partial and fully filled order fee
func collectTradeForPublish(
	tradesToPublish *[]*Trade,
	wg *sync.WaitGroup,
	height int64,
	tradeHolderCh <-chan orderPkg.TradeHolder) {

	defer wg.Done()
	tradeIdx := 0
	trades := make(map[*me.Trade]*Trade)
	for feeHolder := range tradeHolderCh {
		Logger.Debug("processing TradeHolder", "holder", feeHolder.String())
		// one trade has two transfer, we can skip the second
		if _, ok := trades[feeHolder.Trade]; !ok {
			t := &Trade{
				Id:     fmt.Sprintf("%d-%d", height, tradeIdx),
				Symbol: feeHolder.Symbol,
				Sid:    feeHolder.Trade.Sid,
				Bid:    feeHolder.Trade.Bid,
				Price:  feeHolder.Trade.LastPx,
				Qty:    feeHolder.Trade.LastQty}
			trades[feeHolder.Trade] = t
			tradeIdx += 1
			*tradesToPublish = append(*tradesToPublish, t)
		}
	}
}

func updateExpireFeeForPublish(
	dexKeeper *orderPkg.Keeper,
	wg *sync.WaitGroup,
	tranHolderCh <-chan orderPkg.ExpireHolder,
	reason orderPkg.ChangeType) {
	defer wg.Done()
	for tranHolder := range tranHolderCh {
		Logger.Debug("transfer collector for order", "orderId", tranHolder.OrderId)
		change := orderPkg.OrderChange{tranHolder.OrderId, reason}
		dexKeeper.OrderChanges = append(dexKeeper.OrderChanges, change)
	}
}

// collect all changed books according to published order status
func filterChangedOrderBooksByOrders(
	ordersToPublish []order,
	latestPriceLevels orderPkg.ChangedPriceLevelsMap) orderPkg.ChangedPriceLevelsMap {
	var res = make(orderPkg.ChangedPriceLevelsMap)
	// map from symbol -> price -> qty diff in this block
	var buyQtyDiff = make(map[string]map[int64]int64)
	var sellQtyDiff = make(map[string]map[int64]int64)
	var allSymbols = make(map[string]struct{})
	for _, o := range ordersToPublish {
		price := o.price
		symbol := o.symbol

		if _, ok := latestPriceLevels[symbol]; !ok {
			continue
		}
		allSymbols[symbol] = struct{}{}
		if _, ok := res[symbol]; !ok {
			res[symbol] = orderPkg.ChangedPriceLevelsPerSymbol{make(map[int64]int64), make(map[int64]int64)}
			buyQtyDiff[symbol] = make(map[int64]int64)
			sellQtyDiff[symbol] = make(map[int64]int64)
		}

		switch o.side {
		case orderPkg.Side.BUY:
			if qty, ok := latestPriceLevels[symbol].Buys[price]; ok {
				res[symbol].Buys[price] = qty
			} else {
				res[symbol].Buys[price] = 0
			}
			buyQtyDiff[symbol][price] += o.effectQtyToOrderBook()
		case orderPkg.Side.SELL:
			if qty, ok := latestPriceLevels[symbol].Sells[price]; ok {
				res[symbol].Sells[price] = qty
			} else {
				res[symbol].Sells[price] = 0
			}
			sellQtyDiff[symbol][price] += o.effectQtyToOrderBook()
		}
	}

	// filter touched but qty actually not changed price levels
	for symbol, priceToQty := range buyQtyDiff {
		for price, qty := range priceToQty {
			if qty == 0 {
				delete(res[symbol].Buys, price)
			}
		}
	}
	for symbol, priceToQty := range sellQtyDiff {
		for price, qty := range priceToQty {
			if qty == 0 {
				delete(res[symbol].Sells, price)
			}
		}
	}
	for symbol, _ := range allSymbols {
		if len(res[symbol].Sells) == 0 && len(res[symbol].Buys) == 0 {
			delete(res, symbol)
		}
	}

	return res
}

func tradeToOrder(t *Trade, o *orderPkg.OrderInfo, timestamp int64, feeHolder orderPkg.FeeHolder) order {
	var status orderPkg.ChangeType
	if o.CumQty == o.Quantity {
		status = orderPkg.FullyFill
	} else {
		status = orderPkg.PartialFill
	}
	fee := getSerializedFeeForOrder(o, status, feeHolder)
	res := order{
		o.Symbol,
		status,
		o.Id,
		t.Id,
		o.Sender.String(),
		o.Side,
		orderPkg.OrderType.LIMIT,
		o.Price,
		o.Quantity,
		t.Price,
		t.Qty,
		o.CumQty,
		fee,
		o.CreatedTimestamp,
		timestamp,
		o.TimeInForce,
		orderPkg.NEW,
		o.TxHash,
	}
	return res
}

// we collect OrderPart here to make matcheng module independent
func collectExecutedOrdersToPublish(
	trades []*Trade,
	orderChanges orderPkg.OrderChanges,
	orderChangesMap orderPkg.OrderInfoForPublish,
	feeHolder orderPkg.FeeHolder,
	timestamp int64) (opensToPublish []order, canceledToPublish []order) {
	opensToPublish = make([]order, 0)
	canceledToPublish = make([]order, 0)

	// collect orders (new, cancel, ioc-no-fill, expire) from orderChanges
	for _, o := range orderChanges {
		orderInfo := orderChangesMap[o.Id]

		orderToPublish := order{
			orderInfo.Symbol,
			o.Tpe,
			o.Id,
			"",
			orderInfo.Sender.String(),
			orderInfo.Side,
			orderPkg.OrderType.LIMIT,
			orderInfo.Price,
			orderInfo.Quantity,
			0,
			0,
			orderInfo.CumQty,
			getSerializedFeeForOrder(orderInfo, o.Tpe, feeHolder),
			orderInfo.CreatedTimestamp,
			timestamp,
			orderInfo.TimeInForce,
			orderPkg.NEW,
			orderInfo.TxHash,
		}
		if o.Tpe == orderPkg.Ack {
			opensToPublish = append(opensToPublish, orderToPublish)
		} else {
			canceledToPublish = append(canceledToPublish, orderToPublish)
		}
	}

	// update fee and collect orders from trades
	for _, t := range trades {
		if o, exists := orderChangesMap[t.Bid]; exists {
			opensToPublish = append(opensToPublish, tradeToOrder(t, o, timestamp, feeHolder))
		} else {
			Logger.Error("failed to resolve order information from orderChangesMap", "orderId", t.Bid)
		}

		if o, exists := orderChangesMap[t.Sid]; exists {
			opensToPublish = append(opensToPublish, tradeToOrder(t, o, timestamp, feeHolder))
		} else {
			Logger.Error("failed to resolve order information from orderChangesMap", "orderId", t.Sid)
		}
	}

	return opensToPublish, canceledToPublish
}

func getSerializedFeeForOrder(orderInfo *orderPkg.OrderInfo, status orderPkg.ChangeType, feeHolder orderPkg.FeeHolder) string {
	feeStr := ""
	if fee, ok := feeHolder[string(orderInfo.Sender)]; ok {
		feeStr = fee.String()
	} else {
		if orderInfo.CumQty == 0 && status != orderPkg.Ack {
			Logger.Error("cannot find fee from fee holder", "orderId", orderInfo.Id)
		}
	}
	return feeStr
}
