package types

import "math"

type TradingPair struct {
	TradeAsset string `json:"trade_asset"`
	QuoteAsset string `json:"quote_asset"`
	Price      int64  `json:"price"`
	TickSize   int64  `json:"tick_size"`
	LotSize    int64  `json:"lot_size"`
}

func NewTradingPair(tradeAsset, quoteAsset string, price int64) TradingPair {
	return TradingPair{
		TradeAsset: tradeAsset,
		QuoteAsset: quoteAsset,
		Price:      price,
		TickSize:   CalcTickSize(price),
		LotSize:    CalcLotSize(price),
	}
}

// CalcTickSize calculate TickSize based on price
func CalcTickSize(price int64) int64 {
	if price <= 0 {
		return 1
	}

	priceDigits := int64(math.Floor(math.Log10(float64(price))))
	return int64(math.Pow(10, math.Max(float64(priceDigits-5), 0)))
}

// CalcLotSize calculate LotSize based on price
func CalcLotSize(price int64) int64 {
	if price <= 0 {
		return 1e8
	}

	priceDigits := int64(math.Floor(math.Log10(float64(price))))
	return int64(math.Pow(10, math.Max(float64(8-math.Max(float64(priceDigits-5), 0)), 0)))
}
