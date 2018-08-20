package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/utils"
)

func UpdateTickSizeAndLotSize(ctx sdk.Context, tradingPairMapper TradingPairMapper, orderKeeper OrderKeeper) {
	tradingPairs := tradingPairMapper.ListAllTradingPairs(ctx)

	for _, pair := range tradingPairs {
		symbol := utils.Ccy2TradeSymbol(pair.TradeAsset, pair.QuoteAsset)
		_, lastPrice := orderKeeper.GetLastTrades(symbol)
		if lastPrice == 0 {
			continue
		}

		tradingPairMapper.UpdateTickSizeAndLotSize(ctx, pair, lastPrice)
	}
}
