package dex

import (
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
)

const (
	ListingFee = 1e12
)

func init() {
	tx.RegisterCalculator(list.Route, tx.FixedFeeCalculator(ListingFee, types.FeeForAll))
	tx.RegisterCalculator(order.NewOrder, tx.FreeFeeCalculator())
	// the calculation of cancel fee is complicated and similar like expire orders.
	// So set free here and put the real calc in the handler.
	tx.RegisterCalculator(order.CancelOrder, tx.FreeFeeCalculator())
}
