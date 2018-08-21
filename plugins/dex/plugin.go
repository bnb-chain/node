package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/utils"
)

func EndBreatheBlock(ctx sdk.Context, tradingPairMapper TradingPairMapper,
	accountMapper auth.AccountMapper, dexKeeper DexKeeper, height, blockTime int64) {
	updateTickSizeAndLotSize(ctx, tradingPairMapper, dexKeeper)
	dexKeeper.ExpireOrders(ctx, height, accountMapper)
	dexKeeper.MarkBreatheBlock(ctx, height, blockTime)
	dexKeeper.SnapShotOrderBook(ctx, height)
}

func updateTickSizeAndLotSize(ctx sdk.Context, tradingPairMapper TradingPairMapper, dexKeeper DexKeeper) {
	tradingPairs := tradingPairMapper.ListAllTradingPairs(ctx)

	for _, pair := range tradingPairs {
		symbol := utils.Ccy2TradeSymbol(pair.TradeAsset, pair.QuoteAsset)
		_, lastPrice := dexKeeper.GetLastTrades(symbol)
		if lastPrice == 0 {
			continue
		}

		tradingPairMapper.UpdateTickSizeAndLotSize(ctx, pair, lastPrice)
	}
}
