package dex

import (
	"github.com/BiJie/BinanceChain/common/fees"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
)

const (
	ListingFee = 1e12
)

func init() {
	fees.RegisterCalculator(list.Route, fees.FixedFeeCalculator(ListingFee, types.FeeForAll))
	fees.RegisterCalculator(order.RouteNewOrder, fees.FreeFeeCalculator())
	// the calculation of cancel fee is complicated and similar like expire orders.
	// So set free here and put the real calc in the handler.
	fees.RegisterCalculator(order.RouteCancelOrder, fees.FreeFeeCalculator())
}
