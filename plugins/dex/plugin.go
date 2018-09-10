package dex

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/app/pub"
	bnclog "github.com/BiJie/BinanceChain/common/log"
	app "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
)

const abciQueryPrefix = "dex"

// InitPlugin initializes the dex plugin.
func InitPlugin(appp app.ChainApp, keeper *DexKeeper) {
	handler := createQueryHandler(keeper)
	appp.RegisterQueryHandler(abciQueryPrefix, handler)
}

func createQueryHandler(keeper *DexKeeper) app.AbciQueryHandler {
	return createAbciQueryHandler(keeper)
}

// EndBreatheBlock processes the breathe block lifecycle event.
func EndBreatheBlock(ctx sdk.Context, accountMapper auth.AccountMapper, dexKeeper *DexKeeper, height, blockTime int64) {
	logger := bnclog.With("module", "dex")
	logger.Info("Start updating tick size / lot size")
	updateTickSizeAndLotSize(ctx, dexKeeper)
	logger.Info("Staring Expiring stale orders")
	if dexKeeper.CollectOrderInfoForPublish {
		transCh := make(chan order.Transfer, pub.FeeCollectionChannelSize)

		var feeCollectorForTrades = func(tran order.Transfer) {
			transCh <- tran
		}

		dexKeeper.ExpireOrders(ctx, blockTime, accountMapper, feeCollectorForTrades)
		close(transCh)

		for tran := range transCh {
			logger.Debug(fmt.Sprintf("fee Collector for tran: %s", tran.String()))

			var id string
			if tran.IsBuyer() {
				id = tran.Bid
			} else {
				id = tran.Sid
			}
			originOrd := dexKeeper.OrderChangesMap[id]
			var fee int64
			var feeAsset string
			if !tran.FeeFree() {
				fee = tran.Fee.Tokens[0].Amount.Int64() // TODO(#66): Fix potential fee precision loss
				feeAsset = tran.Fee.Tokens[0].Denom
			}
			change := order.OrderChange{
				OrderMsg:  originOrd.OrderMsg,
				Tpe:       order.Expired,
				Fee:       fee,
				FeeAsset:  feeAsset,
				LeavesQty: originOrd.LeavesQty,
				CumQty:    originOrd.CumQty}
			dexKeeper.OrderChanges = append(dexKeeper.OrderChanges, change)
		}
	} else {
		dexKeeper.ExpireOrders(ctx, blockTime, accountMapper, nil)
	}
	logger.Info("Mark BreathBlock", "blockHeight", height)
	dexKeeper.MarkBreatheBlock(ctx, height, blockTime)
	logger.Info("Save Orderbook snapshot", "blockHeight", height)
	if _, err := dexKeeper.SnapShotOrderBook(ctx, height); err != nil {
		logger.Error("Failed to snapshot order book", "blockHeight", height, "err", err)
	}
}

func updateTickSizeAndLotSize(ctx sdk.Context, dexKeeper *DexKeeper) {
	tradingPairs := dexKeeper.PairMapper.ListAllTradingPairs(ctx)

	for _, pair := range tradingPairs {
		_, lastPrice := dexKeeper.GetLastTradesForPair(pair.GetSymbol())
		if lastPrice == 0 {
			continue
		}
		_, lotSize := dexKeeper.PairMapper.UpdateTickSizeAndLotSize(ctx, pair, lastPrice)
		dexKeeper.UpdateLotSize(pair.GetSymbol(), lotSize)
	}
}
