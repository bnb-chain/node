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
	tradesToPublish := make([]Trade, 0)
	go collectTradeForPublish(&tradesToPublish, &wg, ctx.BlockHeader().Height, tradeFeeHolderCh)
	go updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		if !tran.Fee.IsEmpty() {
			// TODO(#160): Fix potential fee precision loss
			fee := orderPkg.Fee{tran.Fee.Tokens[0].Amount.Int64(), tran.Fee.Tokens[0].Denom}
			if tran.IsExpiredWithFee() {
				iocExpireFeeHolderCh <- orderPkg.ExpireFeeHolder{tran.Oid, fee}
			} else {
				tradeFeeHolderCh <- orderPkg.TradeFeeHolder{tran.Oid, tran.Trade, tran.Symbol, fee}
			}
		}
	}
	ctx, _, _ = dexKeeper.MatchAndAllocateAll(ctx, accountMapper, feeCollectorForTrades)
	close(tradeFeeHolderCh)
	close(iocExpireFeeHolderCh)
	wg.Wait()

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
	go updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		// TODO(#160): Fix potential fee precision loss
		fee := orderPkg.Fee{tran.Fee.Tokens[0].Amount.Int64(), tran.Fee.Tokens[0].Denom}
		if tran.IsExpiredWithFee() {
			iocExpireFeeHolderCh <- orderPkg.ExpireFeeHolder{tran.Oid, fee}
		}
	}
	dexKeeper.ExpireOrders(ctx, blockTime, accountMapper, feeCollectorForTrades)
	close(iocExpireFeeHolderCh)
	wg.Wait()
}

// for partial and fully filled order fee
func collectTradeForPublish(
	tradesToPublish *[]Trade,
	wg *sync.WaitGroup,
	height int64,
	feeHolderCh <-chan orderPkg.TradeFeeHolder) {

	defer wg.Done()
	tradeIdx := 0
	groupedTrades := make(map[string]map[string]*Trade)
	for feeHolder := range feeHolderCh {
		Logger.Debug("processing TradeFeeHolder", "feeHolder", feeHolder.String())
		var t *Trade
		// one trade has two transfer, the second fee update should applied to first updated trade
		if groupedByBid, exists := groupedTrades[feeHolder.Trade.Bid]; exists {
			if tradeToPublish, exists := groupedByBid[feeHolder.Trade.Sid]; exists {
				t = tradeToPublish
			} else {
				t = &Trade{Bfee: -1, Sfee: -1} // in case for some orders the fee can be 0
				groupedByBid[feeHolder.Trade.Sid] = t
			}
		} else {
			groupedByBid := make(map[string]*Trade)
			groupedTrades[feeHolder.Trade.Bid] = groupedByBid
			t = &Trade{Bfee: -1, Sfee: -1} // in case for some orders the fee can be 0
			groupedByBid[feeHolder.Trade.Sid] = t
		}

		if feeHolder.OId == feeHolder.Trade.Bid {
			t.Bfee = feeHolder.Amount
			t.BfeeAsset = feeHolder.Asset
		} else {
			t.Sfee = feeHolder.Amount
			t.SfeeAsset = feeHolder.Asset
		}

		if t.Bfee != -1 && t.Sfee != -1 {
			t.Id = fmt.Sprintf("%d-%d", height, tradeIdx)
			t.Symbol = feeHolder.Symbol
			t.Sid = feeHolder.Trade.Sid
			t.Bid = feeHolder.Trade.Bid
			t.Price = feeHolder.Trade.LastPx
			t.Qty = feeHolder.Trade.LastQty
			*tradesToPublish = append(*tradesToPublish, *t)
			tradeIdx += 1
		}
	}
}

func updateExpireFeeForPublish(
	dexKeeper *orderPkg.Keeper,
	wg *sync.WaitGroup,
	feeHolderCh <-chan orderPkg.ExpireFeeHolder) {
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
}
