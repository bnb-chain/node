package types

import (
	ctuils "github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/utils"
)

type TradingPair struct {
	TradeAsset string `json:"trade_asset"`
	QuoteAsset string `json:"quote_asset"`
	Price      int64  `json:"price"`
	TickSize   int64  `json:"tick_size"`
	LotSize    int64  `json:"lot_size"`
}

func NewTradingPair(tradeAsset, quoteAsset string, price int64) TradingPair {
	tickSize, lotSize := utils.CalcTickSizeAndLotSize(price)

	return TradingPair{
		TradeAsset: tradeAsset,
		QuoteAsset: quoteAsset,
		Price:      price,
		TickSize:   tickSize,
		LotSize:    lotSize,
	}
}

func (pair *TradingPair) GetSymbol() string {
	return ctuils.Ccy2TradeSymbol(pair.TradeAsset, pair.QuoteAsset)
}
