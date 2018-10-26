package dex

import (
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
)

const (
	ListingFee     = 3e13
	OrderCancelFee = 1e6
)

func init() {
	tx.RegisterCalculator(list.Route, tx.FixedFeeCalculator(ListingFee, types.FeeForAll))
	tx.RegisterCalculator(order.RouteNewOrder, tx.FreeFeeCalculator())
	tx.RegisterCalculator(order.RouteCancelOrder, tx.FixedFeeCalculator(OrderCancelFee, types.FeeForProposer))
}
