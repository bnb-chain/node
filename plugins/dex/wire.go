package dex

import (
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/cosmos/cosmos-sdk/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(order.MakeOfferMsg{}, "cool/MakeOffer", nil)
	cdc.RegisterConcrete(order.FillOfferMsg{}, "cool/FillOffer", nil)
	cdc.RegisterConcrete(order.CancelOfferMsg{}, "cool/CancelOffer", nil)

	cdc.RegisterConcrete(list.Msg{}, "dex/ListMsg", nil)
	cdc.RegisterConcrete(types.TradingPair{}, "dex/TradingPair", nil)
}
