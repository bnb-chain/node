package minitokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/plugins/minitokens/issue"
	"github.com/binance-chain/node/plugins/minitokens/seturi"
	"github.com/binance-chain/node/plugins/minitokens/store"
)

func Routes(tokenMapper store.MiniTokenMapper, accKeeper auth.AccountKeeper, keeper bank.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, keeper)
	routes[seturi.SetURIRoute] = seturi.NewHandler(tokenMapper)
	return routes
}
