package pub

import (
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

func MatchAndAllocateAllForPublish(
	dexKeeper *orderPkg.Keeper,
	accountMapper auth.AccountMapper,
	ctx sdk.Context) []Trade {
	tradeFeeHolderCh := make(chan orderPkg.TradeFeeHolder, FeeCollectionChannelSize)
	iocExpireFeeHolderCh := make(chan orderPkg.ExpireFeeHolder, FeeCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(2)

	// group trades by Bid and Sid to make fee update easier
	groupedTrades := make(map[string]map[string]Trade)
	collectTradeForPublish(groupedTrades, &wg, tradeFeeHolderCh)
	updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		if !tran.FeeFree() {
			// TODO(#160): Fix potential fee precision loss
			fee := orderPkg.Fee{tran.Fee.Tokens[0].Amount.Int64(), tran.Fee.Tokens[0].Denom}
			if tran.IsExpiredWithFee() {
				iocExpireFeeHolderCh <- orderPkg.ExpireFeeHolder{tran.Bid, fee}
			} else {
				var side int8
				if tran.IsBuyer() {
					side = orderPkg.Side.BUY
				} else {
					side = orderPkg.Side.SELL
				}
				tradeFeeHolderCh <- orderPkg.TradeFeeHolder{tran.Sid, tran.Bid, side, fee}
			}
		}
	}
	ctx, _, _ = dexKeeper.MatchAndAllocateAll(ctx, accountMapper, feeCollectorForTrades)
	close(tradeFeeHolderCh)
	close(iocExpireFeeHolderCh)
	wg.Wait()

	tradeIdx := 0
	allTrades := dexKeeper.GetLastTrades()
	tradesToPublish := make([]Trade, 0)
	for symbol, trades := range *allTrades {
		for _, matchTrade := range trades {
			Logger.Debug("processing trade", "bid", matchTrade.BId, "sid", matchTrade.SId)
			if groupedByBid, exists := groupedTrades[matchTrade.BId]; exists {
				if t, exists := groupedByBid[matchTrade.SId]; exists {
					t.Id = fmt.Sprintf("%d-%d", ctx.BlockHeader().Height, tradeIdx)
					t.Symbol = symbol
					t.Price = matchTrade.LastPx
					t.Qty = matchTrade.LastQty
					tradesToPublish = append(tradesToPublish, t)
					tradeIdx += 1
				} else {
					Logger.Error("failed to look up sid from trade from groupedTrades",
						"bid", matchTrade.BId, "sid", matchTrade.SId)
				}
			} else {
				Logger.Error("failed to look up bid from trade from groupedTrades",
					"bid", matchTrade.BId, "sid", matchTrade.SId)
			}
		}
	}

	return tradesToPublish
}

func ExpireOrdersForPublish(
	dexKeeper *orderPkg.Keeper,
	accountMapper auth.AccountMapper,
	ctx sdk.Context,
	blockTime int64) {
	iocExpireFeeHolderCh := make(chan orderPkg.ExpireFeeHolder, FeeCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(1)
	updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		// TODO(#160): Fix potential fee precision loss
		fee := orderPkg.Fee{tran.Fee.Tokens[0].Amount.Int64(), tran.Fee.Tokens[0].Denom}
		if tran.IsExpiredWithFee() {
			iocExpireFeeHolderCh <- orderPkg.ExpireFeeHolder{tran.Bid, fee}
		}
	}
	dexKeeper.ExpireOrders(ctx, blockTime, accountMapper, feeCollectorForTrades)
	close(iocExpireFeeHolderCh)
	wg.Wait()
}

func collectTradeForPublish(
	groupedTrades map[string]map[string]Trade,
	wg *sync.WaitGroup,
	feeHolderCh <-chan orderPkg.TradeFeeHolder) {

	go func() {
		defer wg.Done()
		for feeHolder := range feeHolderCh {
			Logger.Debug("processing TradeFeeHolder", "feeHolder", feeHolder.String())
			// for partial and fully filled order fee
			var t Trade
			if groupedByBid, exists := groupedTrades[feeHolder.BId]; exists {
				if tradeToPublish, exists := groupedByBid[feeHolder.SId]; exists {
					t = tradeToPublish
				} else {
					t = Trade{}
					t.Sid = feeHolder.SId
					t.Bid = feeHolder.BId
					groupedByBid[feeHolder.SId] = t
				}
			} else {
				groupedByBid := make(map[string]Trade)
				groupedTrades[feeHolder.BId] = groupedByBid
				t = Trade{}
				t.Sid = feeHolder.SId
				t.Bid = feeHolder.BId
				groupedByBid[feeHolder.SId] = t
			}
			if feeHolder.Side == orderPkg.Side.BUY {
				t.Bfee = feeHolder.Amount
				t.BfeeAsset = feeHolder.Asset
			} else {
				t.Sfee = feeHolder.Amount
				t.SfeeAsset = feeHolder.Asset
			}
		}
	}()
}

func updateExpireFeeForPublish(
	dexKeeper *orderPkg.Keeper,
	wg *sync.WaitGroup,
	feeHolderCh <-chan orderPkg.ExpireFeeHolder) {
	go func() {
		defer wg.Done()
		for feeHolder := range feeHolderCh {
			Logger.Debug("fee Collector for expire transfer", "transfer", feeHolder.String())

			id := feeHolder.OrderId
			originOrd := dexKeeper.OrderChangesMap[id]
			var fee int64
			var feeAsset string
			fee = feeHolder.Amount
			feeAsset = feeHolder.Asset
			change := orderPkg.OrderChange{originOrd.Id, orderPkg.Expired, fee, feeAsset}
			dexKeeper.OrderChanges = append(dexKeeper.OrderChanges, change)
		}
	}()
}
