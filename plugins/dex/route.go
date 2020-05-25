package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"
)

// Routes exports dex message routes
func Routes(cdc *wire.Codec, dexKeeper *DexKeeper, tokenMapper tokens.Mapper,
	accKeeper auth.AccountKeeper, govKeeper gov.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	orderHandler := order.NewHandler(cdc, dexKeeper)
	routes[order.RouteNewOrder] = orderHandler
	routes[order.RouteCancelOrder] = orderHandler
	routes[list.Route] = list.NewHandler(dexKeeper, tokenMapper, govKeeper)
	return routes
}
