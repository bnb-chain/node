package types

import (
	ctuils "github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/utils"
)

type TradingPair struct {
	BaseAssetSymbol  string        `json:"base_asset_symbol"`
	QuoteAssetSymbol string        `json:"quote_asset_symbol"`
	ListPrice        ctuils.Fixed8 `json:"list_price"`
	TickSize         ctuils.Fixed8 `json:"tick_size"`
	LotSize          ctuils.Fixed8 `json:"lot_size"`
}

func NewTradingPair(baseAssetSymbol, quoteAssetSymbol string, listPrice int64) TradingPair {
	tickSize, lotSize := utils.CalcTickSizeAndLotSize(listPrice)

	// TODO: symbol validations should also happen here, but a lot of tests rely on this method.
	//       for now, these checks are done in TradingPairMapper#AddTradingPair.

	return TradingPair{
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: quoteAssetSymbol,
		ListPrice:        ctuils.Fixed8(listPrice),
		TickSize:         ctuils.Fixed8(tickSize),
		LotSize:          ctuils.Fixed8(lotSize),
	}
}

func (pair *TradingPair) GetSymbol() string {
	return ctuils.Assets2TradingPair(pair.BaseAssetSymbol, pair.QuoteAssetSymbol)
}
