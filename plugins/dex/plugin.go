package dex

import (
	"time"

	"github.com/BiJie/BinanceChain/common/account"
	"github.com/BiJie/BinanceChain/common/types"
)

const abciQueryPrefix = "dex"

// InitPlugin initializes the dex plugin.
func InitPlugin(appp types.ChainApp, keeper *DexKeeper) {
	handler := createQueryHandler(keeper)
	appp.RegisterQueryHandler(abciQueryPrefix, handler)
}

func createQueryHandler(keeper *DexKeeper) types.AbciQueryHandler {
	return createAbciQueryHandler(keeper)
}

// EndBreatheBlock processes the breathe block lifecycle event.
func EndBreatheBlock(ctx types.Context, accountMapper account.Mapper, dexKeeper DexKeeper, height int64, blockTime time.Time) {
	updateTickSizeAndLotSize(ctx, dexKeeper)
	dexKeeper.ExpireOrders(ctx, height, accountMapper)
	dexKeeper.MarkBreatheBlock(ctx, height, blockTime)
	dexKeeper.SnapShotOrderBook(ctx, height)
}

func updateTickSizeAndLotSize(ctx types.Context, dexKeeper DexKeeper) {
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
