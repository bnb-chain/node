package tokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func Routes(tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, accountMapper, keeper)
	routes[burn.Route] = burn.NewHandler(tokenMapper, keeper)
	freezeHandler := freeze.NewHandler(tokenMapper, accountMapper, keeper)
	routes[freeze.RouteFreeze] = freezeHandler
	routes[freeze.RouteUnfreeze] = freezeHandler
	return routes
}
