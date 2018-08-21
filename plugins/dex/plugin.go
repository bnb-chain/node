package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/utils"
)

func EndBreatheBlock(ctx sdk.Context, tradingPairMapper TradingPairMapper, dexKeeper DexKeeper) {
	updateTickSizeAndLotSize(ctx, tradingPairMapper, dexKeeper)
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
