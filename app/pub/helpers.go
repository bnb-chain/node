package pub

import (
	"fmt"
	"sync"
	"time"

	"github.com/binance-chain/node/common/types"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

func GetTradeAndOrdersRelatedAccounts(kp *orderPkg.Keeper, tradesToPublish []*Trade) []string {
	res := make([]string, 0, len(tradesToPublish)*2+len(kp.OrderChanges))

	for _, t := range tradesToPublish {
		if bo, ok := kp.OrderInfosForPub[t.Bid]; ok {
			res = append(res, string(bo.Sender.Bytes()))
		} else {
			Logger.Error("failed to locate buy order in OrderChangesMap for trade account resolving", "bid", t.Bid)
		}
		if so, ok := kp.OrderInfosForPub[t.Sid]; ok {
			res = append(res, string(so.Sender.Bytes()))
		} else {
			Logger.Error("failed to locate sell order in OrderChangesMap for trade account resolving", "sid", t.Sid)
		}
	}

	for _, orderChange := range kp.OrderChanges {
		if orderInfo := kp.OrderInfosForPub[orderChange.Id]; orderInfo != nil {
			res = append(res, string(orderInfo.Sender.Bytes()))
		} else {
			Logger.Error("failed to locate order change in OrderChangesMap", "orderChange", orderChange.String())
		}
	}

	return res
}

func GetTransferPublished(pool *sdk.Pool, height, blockTime int64) *Transfers {
	transferToPublish := make([]Transfer, 0, 0)
	txs := pool.GetTxs()
	txs.Range(func(key, value interface{}) bool {
		txhash := key.(string)
		t := value.(sdk.Tx)
		msgs := t.GetMsgs()
		for _, m := range msgs {
			msg, ok := m.(bank.MsgSend)
			if !ok {
				continue
			}
			receivers := make([]Receiver, 0, len(msg.Outputs))
			for _, o := range msg.Outputs {
				coins := make([]Coin, 0, len(o.Coins))
				for _, c := range o.Coins {
					coins = append(coins, Coin{c.Denom, c.Amount})
				}
				receivers = append(receivers, Receiver{Addr: o.Address.String(), Coins: coins})
			}
			transferToPublish = append(transferToPublish, Transfer{TxHash: txhash, From: msg.Inputs[0].Address.String(), To: receivers})
		}
		return true
	})
	return &Transfers{Height: height, Num: len(transferToPublish), Timestamp: blockTime, Transfers: transferToPublish}
}

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
							assetBalance.Free = freeCoin.Amount
						} else {
							newAB := &AssetBalance{Asset: freeCoin.Denom, Free: freeCoin.Amount}
							assets = append(assets, newAB)
							assetsMap[freeCoin.Denom] = newAB
						}
					}

					for _, frozenCoin := range acc.GetFrozenCoins() {
						if assetBalance, ok := assetsMap[frozenCoin.Denom]; ok {
							assetBalance.Frozen = frozenCoin.Amount
						} else {
							newAB := &AssetBalance{Asset: frozenCoin.Denom, Frozen: frozenCoin.Amount}
							assets = append(assets, newAB)
							assetsMap[frozenCoin.Denom] = newAB
						}
					}

					for _, lockedCoin := range acc.GetLockedCoins() {
						if assetBalance, ok := assetsMap[lockedCoin.Denom]; ok {
							assetBalance.Locked = lockedCoin.Amount
						} else {
							newAB := &AssetBalance{Asset: lockedCoin.Denom, Locked: lockedCoin.Amount}
							assets = append(assets, newAB)
							assetsMap[lockedCoin.Denom] = newAB
						}
					}

					res[addrBytesStr] = Account{Owner: addrBytesStr, Balances: assets}
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
	ctx sdk.Context) ([]*Trade, []*Combination) {
	// These two channels are used for protect not update `tradesToPublish` and `dexKeeper.OrderChanges` concurrently
	// matcher would send item to feeCollectorForTrades in several goroutine (well-designed)
	// while tradesToPublish and dexKeeper.OrderChanges are not separated by concurrent factor (users here), so we have
	// to organized transfer holders into 2 channels
	tradeHolderCh := make(chan orderPkg.TradeHolder, TransferCollectionChannelSize)
	iocExpireFeeHolderCh := make(chan orderPkg.ExpireHolder, TransferCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(2)

	engineSurplus := make(map[string]int64)
	tradesToPublish := make([]*Trade, 0)
	go collectTradeForPublish(&tradesToPublish, &wg, ctx.BlockHeader().Height, tradeHolderCh)
	go updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		if tran.IsExpire() {
			if tran.IsExpiredWithFee() {
				// we only got expire of Ioc here, gte orders expire is handled in breathe block
				iocExpireFeeHolderCh <- orderPkg.ExpireHolder{tran.Oid, orderPkg.IocNoFill}
			} else {
				iocExpireFeeHolderCh <- orderPkg.ExpireHolder{tran.Oid, orderPkg.IocExpire}
			}
		} else {
			tradeHolderCh <- orderPkg.TradeHolder{tran.Oid, tran.Trade, tran.Symbol}
		}
	}

	dexKeeper.MatchAndAllocateAll(ctx, feeCollectorForTrades, engineSurplus)
	close(tradeHolderCh)
	close(iocExpireFeeHolderCh)
	wg.Wait()
	var combinations []*Combination
	for symbol, surplus := range engineSurplus {
		combinations = append(combinations, &Combination{
			Symbol:  symbol,
			Surplus: surplus,
		})
	}

	return tradesToPublish, combinations
}

func ExpireOrdersForPublish(
	dexKeeper *orderPkg.Keeper,
	ctx sdk.Context,
	blockTime time.Time) {
	expireHolderCh := make(chan orderPkg.ExpireHolder, TransferCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go updateExpireFeeForPublish(dexKeeper, &wg, expireHolderCh)
	var collectorForExpires = func(tran orderPkg.Transfer) {
		if tran.IsExpire() {
			expireHolderCh <- orderPkg.ExpireHolder{tran.Oid, orderPkg.Expired}
		}
	}
	dexKeeper.ExpireOrders(ctx, blockTime, collectorForExpires)
	close(expireHolderCh)
	wg.Wait()
	return
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
	for tradeHolder := range tradeHolderCh {
		Logger.Debug("processing TradeHolder", "holder", tradeHolder.String())
		// one trade has two transfer, we can skip the second
		if _, ok := trades[tradeHolder.Trade]; !ok {
			t := &Trade{
				Id:     fmt.Sprintf("%d-%d", height, tradeIdx),
				Symbol: tradeHolder.Symbol,
				Sid:    tradeHolder.Trade.Sid,
				Bid:    tradeHolder.Trade.Bid,
				Price:  tradeHolder.Trade.LastPx,
				Qty:    tradeHolder.Trade.LastQty}
			trades[tradeHolder.Trade] = t
			tradeIdx += 1
			*tradesToPublish = append(*tradesToPublish, t)
		}
	}
}

func CollectProposalsForPublish(passed, failed []int64) Proposals {
	totalProposals := len(passed) + len(failed)
	ps := make([]*Proposal, 0, totalProposals)
	for _, p := range passed {
		ps = append(ps, &Proposal{p, Succeed})
	}
	for _, p := range failed {
		ps = append(ps, &Proposal{p, Failed})
	}
	return Proposals{totalProposals, ps}
}

func CollectStakeUpdatesForPublish(unbondingDelegations []stake.UnbondingDelegation) StakeUpdates {
	length := len(unbondingDelegations)
	completedUnbondingDelegations := make([]*CompletedUnbondingDelegation, 0, length)
	for _, ubd := range unbondingDelegations {
		amount := Coin{ubd.Balance.Denom, ubd.Balance.Amount}
		completedUnbondingDelegations = append(completedUnbondingDelegations, &CompletedUnbondingDelegation{ubd.ValidatorAddr, ubd.DelegatorAddr, amount})
	}
	return StakeUpdates{length, completedUnbondingDelegations}
}

func CollectCombinationsSurplusForPublish(combinations []*Combination) CombinationsSurplus {
	length := len(combinations)
	return CombinationsSurplus{length, combinations}
}

func updateExpireFeeForPublish(
	dexKeeper *orderPkg.Keeper,
	wg *sync.WaitGroup,
	tranHolderCh <-chan orderPkg.ExpireHolder) {
	defer wg.Done()
	for tranHolder := range tranHolderCh {
		Logger.Debug("transfer collector for order", "orderId", tranHolder.OrderId)
		change := orderPkg.OrderChange{tranHolder.OrderId, tranHolder.Reason, nil}
		dexKeeper.OrderChanges = append(dexKeeper.OrderChanges, change)
	}
}

// collect all changed books according to published order status
func filterChangedOrderBooksByOrders(
	ordersToPublish []*Order,
	latestPriceLevels orderPkg.ChangedPriceLevelsMap) orderPkg.ChangedPriceLevelsMap {
	var res = make(orderPkg.ChangedPriceLevelsMap)
	// map from symbol -> price -> qty diff in this block
	var buyQtyDiff = make(map[string]map[int64]int64)
	var sellQtyDiff = make(map[string]map[int64]int64)
	var allSymbols = make(map[string]struct{})
	for _, o := range ordersToPublish {
		price := o.Price
		symbol := o.Symbol

		if _, ok := latestPriceLevels[symbol]; !ok {
			continue
		}
		allSymbols[symbol] = struct{}{}
		if _, ok := res[symbol]; !ok {
			res[symbol] = orderPkg.ChangedPriceLevelsPerSymbol{make(map[int64]int64), make(map[int64]int64)}
			buyQtyDiff[symbol] = make(map[int64]int64)
			sellQtyDiff[symbol] = make(map[int64]int64)
		}

		switch o.Side {
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

func tradeToOrder(t *Trade, o *orderPkg.OrderInfo, timestamp int64, feeHolder orderPkg.FeeHolder, feeToPublish map[string]string) Order {
	var status orderPkg.ChangeType
	if o.CumQty == o.Quantity {
		status = orderPkg.FullyFill
	} else {
		status = orderPkg.PartialFill
	}
	fee := getSerializedFeeForOrder(o, status, feeHolder, feeToPublish)
	owner := o.Sender
	res := Order{
		o.Symbol,
		status,
		o.Id,
		t.Id,
		owner.String(),
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
	if o.Side == orderPkg.Side.BUY {
		t.BAddr = string(owner.Bytes())
		t.Bfee = fee
	} else {
		t.SAddr = string(owner.Bytes())
		t.Sfee = fee
	}
	return res
}

// we collect OrderPart here to make matcheng module independent
func collectOrdersToPublish(
	trades []*Trade,
	orderChanges orderPkg.OrderChanges,
	orderInfos orderPkg.OrderInfoForPublish,
	feeHolder orderPkg.FeeHolder,
	timestamp int64) (opensToPublish []*Order, closedToPublish []*Order, feeToPublish map[string]string) {
	opensToPublish = make([]*Order, 0)
	closedToPublish = make([]*Order, 0)
	// serve as a cache to avoid fee's serialization several times for one address
	feeToPublish = make(map[string]string)

	// the following two maps are used to update fee field we published
	// more detail can be found at:
	// https://github.com/binance-chain/docs-site/wiki/Fee-Calculation,-Collection-and-Distribution#publication
	chargedCancels := make(map[string]int)
	chargedExpires := make(map[string]int)

	// collect orders (new, cancel, ioc-no-fill, expire, failed-blocking and failed-matching) from orderChanges
	for _, o := range orderChanges {
		if orderInfo := o.ResolveOrderInfo(orderInfos); orderInfo != nil {
			orderToPublish := Order{
				orderInfo.Symbol, o.Tpe, o.Id,
				"", orderInfo.Sender.String(), orderInfo.Side,
				orderPkg.OrderType.LIMIT, orderInfo.Price, orderInfo.Quantity,
				0, 0, orderInfo.CumQty, "",
				orderInfo.CreatedTimestamp, timestamp, orderInfo.TimeInForce,
				orderPkg.NEW, orderInfo.TxHash,
			}

			if o.Tpe.IsOpen() {
				opensToPublish = append(opensToPublish, &orderToPublish)
			} else {
				closedToPublish = append(closedToPublish, &orderToPublish)
			}

			// fee field handling
			if orderToPublish.isChargedCancel() {
				if _, ok := chargedCancels[string(orderInfo.Sender)]; ok {
					chargedCancels[string(orderInfo.Sender)] += 1
				} else {
					chargedCancels[string(orderInfo.Sender)] = 1
				}
			} else if orderToPublish.isChargedExpire() {
				if _, ok := chargedExpires[string(orderInfo.Sender)]; ok {
					chargedExpires[string(orderInfo.Sender)] += 1
				} else {
					chargedExpires[string(orderInfo.Sender)] = 1
				}
			}
		} else {
			Logger.Error("failed to locate order change in OrderChangesMap", "orderChange", o.String())
		}
	}

	// update C and E fields in serialized fee string
	for _, order := range closedToPublish {
		if orderInfo, ok := orderInfos[order.OrderId]; ok {
			senderBytesStr := string(orderInfo.Sender)
			if _, ok := feeToPublish[senderBytesStr]; !ok {
				numOfChargedCanceled := chargedCancels[senderBytesStr]
				numOfExpiredCanceled := chargedExpires[senderBytesStr]
				if raw, ok := feeHolder[senderBytesStr]; ok {
					fee := raw.SerializeForPub(numOfChargedCanceled, numOfExpiredCanceled)
					feeToPublish[senderBytesStr] = fee
					order.Fee = fee
				} else if numOfChargedCanceled > 0 || numOfExpiredCanceled > 0 {
					Logger.Error("cannot find fee for cancel/expire", "sender", order.Owner)
				}
			}
		} else {
			Logger.Error("should not to locate order in OrderChangesMap", "oid", order.OrderId)
		}
	}

	// update fee and collect orders from trades
	for _, t := range trades {
		if o, exists := orderInfos[t.Bid]; exists {
			orderToPublish := tradeToOrder(t, o, timestamp, feeHolder, feeToPublish)
			if orderToPublish.Status.IsOpen() {
				opensToPublish = append(opensToPublish, &orderToPublish)
			} else {
				closedToPublish = append(closedToPublish, &orderToPublish)
			}
		} else {
			Logger.Error("failed to resolve order information from orderInfos", "orderId", t.Bid)
		}

		if o, exists := orderInfos[t.Sid]; exists {
			orderToPublish := tradeToOrder(t, o, timestamp, feeHolder, feeToPublish)
			if orderToPublish.Status.IsOpen() {
				opensToPublish = append(opensToPublish, &orderToPublish)
			} else {
				closedToPublish = append(closedToPublish, &orderToPublish)
			}
		} else {
			Logger.Error("failed to resolve order information from orderInfos", "orderId", t.Sid)
		}
	}

	return opensToPublish, closedToPublish, feeToPublish
}

func getSerializedFeeForOrder(orderInfo *orderPkg.OrderInfo, status orderPkg.ChangeType, feeHolder orderPkg.FeeHolder, feeToPublish map[string]string) string {
	senderStr := string(orderInfo.Sender)

	// if the serialized fee has been cached, return it directly
	if cached, ok := feeToPublish[senderStr]; ok {
		return cached
	} else {
		feeStr := ""
		if fee, ok := feeHolder[senderStr]; ok {
			feeStr = fee.String()
			feeToPublish[senderStr] = feeStr
		} else {
			if orderInfo.CumQty == 0 && status != orderPkg.Ack {
				Logger.Error("cannot find fee from fee holder", "orderId", orderInfo.Id)
			}
		}
		return feeStr
	}

}
