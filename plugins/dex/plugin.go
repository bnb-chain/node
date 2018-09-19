package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	bnclog "github.com/BiJie/BinanceChain/common/log"
	app "github.com/BiJie/BinanceChain/common/types"
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
func EndBreatheBlock(ctx sdk.Context, accountMapper auth.AccountMapper, dexKeeper DexKeeper, height, blockTime int64) {
	logger := bnclog.With("module", "dex")
	logger.Info("Start updating tick size / lot size")
	updateTickSizeAndLotSize(ctx, dexKeeper)
	logger.Info("Staring Expiring stale orders")
	dexKeeper.ExpireOrders(ctx, height, accountMapper)
	logger.Info("Mark BreathBlock", "blockHeight", height)
	dexKeeper.MarkBreatheBlock(ctx, height, blockTime)
	logger.Info("Save Orderbook snapshot", "blockHeight", height)
	dexKeeper.SnapShotOrderBook(ctx, height)
}

func updateTickSizeAndLotSize(ctx sdk.Context, dexKeeper DexKeeper) {
	tradingPairs := dexKeeper.PairMapper.ListAllTradingPairs(ctx)

	for _, pair := range tradingPairs {
		_, lastPrice := dexKeeper.GetLastTrades(pair.GetSymbol())
		if lastPrice == 0 {
			continue
		}
		_, lotSize := dexKeeper.PairMapper.UpdateTickSizeAndLotSize(ctx, pair, lastPrice)
		dexKeeper.UpdateLotSize(pair.GetSymbol(), lotSize)
	}
}
