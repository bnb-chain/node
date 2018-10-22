package dex

import (
	"github.com/cosmos/cosmos-sdk/x/auth"

	common "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

// Routes exports dex message routes
func Routes(cdc *wire.Codec, dexKeeper *DexKeeper, tokenMapper tokens.Mapper,
	accountMapper auth.AccountMapper) map[string]common.Handler {
	routes := make(map[string]common.Handler)
	orderHandler := order.NewHandler(cdc, dexKeeper, accountMapper)
	routes[order.NewOrder] = orderHandler
	routes[order.CancelOrder] = orderHandler
	routes[list.Route] = list.NewHandler(dexKeeper, tokenMapper)
	return routes
}
