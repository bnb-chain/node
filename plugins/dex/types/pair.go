package types

import (
	ctuils "github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/utils"
)

type TradingPair struct {
	TradeAsset string        `json:"trade_asset"`
	QuoteAsset string        `json:"quote_asset"`
	Price      ctuils.Fixed8 `json:"price"`
	TickSize   ctuils.Fixed8 `json:"tick_size"`
	LotSize    ctuils.Fixed8 `json:"lot_size"`
}

func NewTradingPair(tradeAsset, quoteAsset string, price int64) TradingPair {
	tickSize, lotSize := utils.CalcTickSizeAndLotSize(price)

	return TradingPair{
		TradeAsset: tradeAsset,
		QuoteAsset: quoteAsset,
		Price:      ctuils.Fixed8(price),
		TickSize:   ctuils.Fixed8(tickSize),
		LotSize:    ctuils.Fixed8(lotSize),
	}
}

func (pair *TradingPair) GetSymbol() string {
	return ctuils.Ccy2TradeSymbol(pair.TradeAsset, pair.QuoteAsset)
}
