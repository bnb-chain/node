package dex

import (
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(Genesis{}, "dex/Genesis", nil)

	cdc.RegisterConcrete(order.NewOrderMsg{}, "dex/NewOrder", nil)
	cdc.RegisterConcrete(order.CancelOrderMsg{}, "dex/CancelOrder", nil)

	cdc.RegisterConcrete(order.NewOrderResponse{}, "dex/NewOrderResponse", nil)

	cdc.RegisterConcrete(list.ListMsg{}, "dex/ListMsg", nil)
	cdc.RegisterConcrete(types.TradingPair{}, "dex/TradingPair", nil)

	cdc.RegisterConcrete(order.OrderBookSnapshot{}, "dex/OrderBookSnapshot", nil)
	cdc.RegisterConcrete(order.ActiveOrders{}, "dex/ActiveOrders", nil)
	cdc.RegisterConcrete(order.TradingGenesis{}, "dex/TradingGenesis", nil)
}
