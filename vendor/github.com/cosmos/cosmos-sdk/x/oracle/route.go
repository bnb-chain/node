package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/oracle/types"
)

func Routes(keeper Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[types.RouteOracle] = NewHandler(keeper)
	return routes
}
