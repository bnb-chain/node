package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

func Routes(tradingPairMapper TradingPairMapper, orderKeeper OrderKeeper, tokenMapper tokens.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	orderHandler := order.NewHandler(orderKeeper, accountMapper)
	routes[order.RouteNew] = orderHandler
	routes[order.RouteCancel] = orderHandler
	routes[list.Route] = list.NewHandler(tradingPairMapper, tokenMapper)
	return routes
}
