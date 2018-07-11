package dex

import (
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Routes(tradingPairMapper TradingPairMapper, orderKeeper OrderKeeper, tokenMapper tokens.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[order.Route] = order.NewHandler(orderKeeper, accountMapper)
	routes[list.Route] = list.NewHandler(tradingPairMapper, tokenMapper)
	return routes
}
