package dex

import (
	"github.com/cosmos/cosmos-sdk/wire"

	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(order.NewOrderMsg{}, "dex/NewOrder", nil)
	cdc.RegisterConcrete(order.CancelOrderMsg{}, "dex/CancelOrder", nil)

	cdc.RegisterConcrete(list.Msg{}, "dex/ListMsg", nil)
	cdc.RegisterConcrete(types.TradingPair{}, "dex/TradingPair", nil)

	cdc.RegisterConcrete(order.TradingGenesis{}, "dex/TradingGenesis", nil)
	cdc.RegisterConcrete(Genesis{}, "dex/Genesis", nil)
}
