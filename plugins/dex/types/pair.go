package types

import (
	ctuils "github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/utils"
)

type TradingPair struct {
	BaseAssetSymbol  string        `json:"base_asset_symbol"`
	QuoteAssetSymbol string        `json:"quote_asset_symbol"`
	Price            ctuils.Fixed8 `json:"price"`
	TickSize         ctuils.Fixed8 `json:"tick_size"`
	LotSize          ctuils.Fixed8 `json:"lot_size"`
}

func NewTradingPair(baseAssetSymbol, quoteAssetSymbol string, price int64) TradingPair {
	tickSize, lotSize := utils.CalcTickSizeAndLotSize(price)

	// TODO: symbol validations should also happen here, but a lot of tests rely on this method.
	//       for now, these checks are done in TradingPairMapper#AddTradingPair.

	return TradingPair{
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: quoteAssetSymbol,
		Price:            ctuils.Fixed8(price),
		TickSize:         ctuils.Fixed8(tickSize),
		LotSize:          ctuils.Fixed8(lotSize),
	}
}

func (pair *TradingPair) GetSymbol() string {
	return utils.Assets2TradingPair(pair.BaseAssetSymbol, pair.QuoteAssetSymbol)
}
