package tokens

import (
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func Routes(tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) map[string]tx.Handler {
	routes := make(map[string]tx.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, keeper)
	routes[burn.Route] = burn.NewHandler(tokenMapper, keeper)
	routes[freeze.RouteFreeze] = freeze.NewHandler(tokenMapper, accountMapper, keeper)
	return routes
}
