package tokens

import (
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/account"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/BiJie/BinanceChain/plugins/tokens/transfer"
)

func Routes(tokenMapper store.Mapper, accountMapper account.Mapper, keeper account.Keeper) map[string]types.Handler {
	routes := make(map[string]types.Handler)
	routes[bank.MsgSend{}.Type()] = transfer.NewHandler(keeper)
	routes[issue.Route] = issue.NewHandler(tokenMapper, keeper)
	routes[burn.Route] = burn.NewHandler(tokenMapper, keeper)
	routes[freeze.RouteFreeze] = freeze.NewHandler(accountMapper, keeper)
	return routes
}
