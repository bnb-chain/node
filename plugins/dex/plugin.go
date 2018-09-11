package dex

import (
	"time"

	"github.com/BiJie/BinanceChain/common/account"
	"github.com/BiJie/BinanceChain/common/types"
)

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
