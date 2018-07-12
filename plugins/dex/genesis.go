package dex

import "github.com/BiJie/BinanceChain/plugins/dex/order"

// TODO: maybe we need other things to put into genesis besides the TradingGenesis
type Genesis struct {
	order.TradingGenesis
}
