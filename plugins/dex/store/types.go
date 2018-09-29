package store

import (
	"github.com/BiJie/BinanceChain/common/utils"
)

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
