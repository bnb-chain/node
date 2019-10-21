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

// only for test use
func NewTradingPair(baseAssetSymbol, quoteAssetSymbol string, listPrice int64) TradingPair {
	lotSize := utils.CalcLotSize(listPrice)
	return NewTradingPairWithLotSize(baseAssetSymbol, quoteAssetSymbol, listPrice, lotSize)
}

func NewTradingPairWithLotSize(baseAsset, quoteAsset string, listPrice, lotSize int64) TradingPair {
	tickSize := utils.CalcTickSize(listPrice)
	return TradingPair{
		BaseAssetSymbol:  baseAsset,
		QuoteAssetSymbol: quoteAsset,
		ListPrice:        ctuils.Fixed8(listPrice),
		TickSize:         ctuils.Fixed8(tickSize),
		LotSize:          ctuils.Fixed8(lotSize),
	}
}

func (pair *TradingPair) GetSymbol() string {
	return utils.Assets2TradingPair(pair.BaseAssetSymbol, pair.QuoteAssetSymbol)
}
