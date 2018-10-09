package store

import (
	"github.com/BiJie/BinanceChain/common/utils"
)

// OrderBook represents an order book at the current point block height, which is included in its struct.
type OrderBook struct {
	Height int64
	Levels []OrderBookLevel
}

// OrderBookLevel represents a single order book level.
type OrderBookLevel struct {
	BuyQty    utils.Fixed8 `json:"buyQty"`
	BuyPrice  utils.Fixed8 `json:"buyPrice"`
	SellQty   utils.Fixed8 `json:"sellQty"`
	SellPrice utils.Fixed8 `json:"sellPrice"`
}
