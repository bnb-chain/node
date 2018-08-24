package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

func Routes(dexKeeper DexKeeper, tokenMapper tokens.Mapper,
	accountMapper auth.AccountMapper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	orderHandler := order.NewHandler(dexKeeper, accountMapper)
	routes[order.NewOrder] = orderHandler
	routes[order.CancelOrder] = orderHandler
	routes[list.Route] = list.NewHandler(dexKeeper, tokenMapper)
	return routes
}
