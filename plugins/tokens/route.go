package tokens

import (
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

func Routes(tokenMapper store.Mapper, keeper bank.CoinKeeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, keeper)
	routes[burn.Route] = burn.NewHandler(tokenMapper, keeper)
	routes[freeze.Route] = freeze.NewHandler(tokenMapper, keeper)
	return routes
}
