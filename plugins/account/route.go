package account

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/binance-chain/node/plugins/account/setaccountflags"
)

func routes(accKeeper auth.AccountKeeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[setaccountflags.AccountFlagsRoute] = setaccountflags.NewHandler(accKeeper)
	return routes
}
