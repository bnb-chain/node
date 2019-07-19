package account

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

func routes(accKeeper auth.AccountKeeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[AccountFlagsRoute] = NewHandler(accKeeper)
	return routes
}
