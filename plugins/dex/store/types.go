package store

import (
	"github.com/BiJie/BinanceChain/common/utils"
)

type Order struct {
	BuyQty    utils.Fixed8 `json:"buyQty"`
	BuyPrice  utils.Fixed8 `json:"buyPrice"`
	SellQty   utils.Fixed8 `json:"sellQty"`
	SellPrice utils.Fixed8 `json:"sellPrice"`
}
