package store

import (
	"github.com/binance-chain/node/common/utils"
)

// OrderBook represents an order book at the current point block height, which is included in its struct.
type OrderBook struct {
	Height       int64
	Levels       []OrderBookLevel
	PendingMatch bool
}

// OrderBookLevel represents a single order book level.
type OrderBookLevel struct {
	BuyQty    utils.Fixed8 `json:"buyQty"`
	BuyPrice  utils.Fixed8 `json:"buyPrice"`
	SellQty   utils.Fixed8 `json:"sellQty"`
	SellPrice utils.Fixed8 `json:"sellPrice"`
}

type OpenOrder struct {
	Id                   string       `json:"id"`
	Symbol               string       `json:"symbol"`
	Price                utils.Fixed8 `json:"price"`
	Quantity             utils.Fixed8 `json:"quantity"`
	CumQty               utils.Fixed8 `json:"cumQty"`
	CreatedHeight        int64        `json:"createdHeight"`
	CreatedTimestamp     int64        `json:"createdTimestamp"`
	LastUpdatedHeight    int64        `json:"lastUpdatedHeight"`
	LastUpdatedTimestamp int64        `json:"lastUpdatedTimestamp"`
}

type RecentPrice struct {
	Pair  []string
	Price []int64
}

func (prices *RecentPrice) removePair(symbolToDelete string) {
	numSymbol := len(prices.Pair)
	for i := 0; i < numSymbol; i++ {
		symbol := prices.Pair[i]
		if symbol == symbolToDelete {
			prices.Pair = append(prices.Pair[:i], prices.Pair[i+1:]...)
			prices.Price = append(prices.Price[:i], prices.Price[i+1:]...)
			break
		}
	}
}
