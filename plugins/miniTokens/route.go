package miniTokens

import (
	"github.com/binance-chain/node/plugins/miniTokens/issue"
	"github.com/binance-chain/node/plugins/miniTokens/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

func Routes(tokenMapper store.MiniTokenMapper, accKeeper auth.AccountKeeper, keeper bank.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, keeper)
	return routes
}
