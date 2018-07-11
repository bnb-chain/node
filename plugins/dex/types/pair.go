package types

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
		TickSize:   calcTickSize(),
		LotSize:    calcLotSize(),
	}
}

// TODO:
func calcTickSize() int64 {
	return 1
}

// TODO:
func calcLotSize() int64 {
	return 1e8
}

func GetPairLabel(tradeAsset, quoteAsset string) string {
	return tradeAsset + "_" + quoteAsset
}
