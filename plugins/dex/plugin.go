package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

func EndBreatheBlock(ctx sdk.Context, accountMapper auth.AccountMapper, dexKeeper DexKeeper, height, blockTime int64) {
	updateTickSizeAndLotSize(ctx, dexKeeper)
	dexKeeper.ExpireOrders(ctx, height, accountMapper)
	dexKeeper.MarkBreatheBlock(ctx, height, blockTime)
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
