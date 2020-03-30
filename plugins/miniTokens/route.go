package miniTokens

import (
	"github.com/binance-chain/node/plugins/miniTokens/burn"
	"github.com/binance-chain/node/plugins/miniTokens/freeze"
	"github.com/binance-chain/node/plugins/miniTokens/issue"
	"github.com/binance-chain/node/plugins/miniTokens/store"
	"github.com/binance-chain/node/plugins/miniTokens/uri"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

func Routes(tokenMapper store.MiniTokenMapper, accKeeper auth.AccountKeeper, keeper bank.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, keeper)
	routes[freeze.FreezeRoute] = freeze.NewHandler(tokenMapper, accKeeper, keeper)
	routes[freeze.FreezeRoute] = freeze.NewHandler(tokenMapper, accKeeper, keeper)
	routes[burn.BurnRoute] = burn.NewHandler(tokenMapper,keeper)
	routes[uri.SetURIRoute] = uri.NewHandler(tokenMapper)
	return routes
}
