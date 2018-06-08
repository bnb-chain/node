package tokens

import (
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

func Routes(tokenMapper store.Mapper, accountMapper sdk.AccountMapper, keeper bank.CoinKeeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, keeper)
	routes[burn.Route] = burn.NewHandler(tokenMapper, keeper)
	freezeHandler := freeze.NewHandler(tokenMapper, accountMapper, keeper)
	routes[freeze.RouteFreeze] = freezeHandler
	routes[freeze.RouteUnfreeze] = freezeHandler
	return routes
}
