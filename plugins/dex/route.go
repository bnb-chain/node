package dex

import (
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/app/router"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

// Routes exports dex message routes
func Routes(cdc *wire.Codec, dexKeeper *DexKeeper, tokenMapper tokens.Mapper,
	accKeeper auth.AccountKeeper) map[string]router.Handler {
	routes := make(map[string]router.Handler)
	orderHandler := order.NewHandler(cdc, dexKeeper, accKeeper)
	routes[order.NewOrder] = orderHandler
	routes[order.CancelOrder] = orderHandler
	routes[list.Route] = list.NewHandler(dexKeeper, tokenMapper)
	return routes
}
