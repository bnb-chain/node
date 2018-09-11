package dex

import (
	"github.com/BiJie/BinanceChain/common/account"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

func Routes(cdc *wire.Codec, dexKeeper DexKeeper, tokenMapper tokens.Mapper,
	accountMapper account.Mapper) map[string]types.Handler {
	routes := make(map[string]types.Handler)
	orderHandler := order.NewHandler(cdc, dexKeeper, accountMapper)
	routes[order.NewOrder] = orderHandler
	routes[order.CancelOrder] = orderHandler
	routes[list.Route] = list.NewHandler(dexKeeper, tokenMapper)
	return routes
}
