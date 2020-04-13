package dex

import (
	"github.com/binance-chain/node/plugins/dex/listmini"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	miniTkstore "github.com/binance-chain/node/plugins/minitokens/store"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"
)

// Routes exports dex message routes
func Routes(cdc *wire.Codec, dexKeeper *DexKeeper, dexMiniKeeper *DexMiniTokenKeeper, tokenMapper tokens.Mapper, miniTokenMapper miniTkstore.MiniTokenMapper,
	accKeeper auth.AccountKeeper, govKeeper gov.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	orderHandler := order.NewHandler(cdc, dexKeeper, dexMiniKeeper, accKeeper)
	routes[order.RouteNewOrder] = orderHandler
	routes[order.RouteCancelOrder] = orderHandler
	routes[list.Route] = list.NewHandler(dexKeeper, tokenMapper, govKeeper)
	routes[listmini.Route] = listmini.NewHandler(dexMiniKeeper, miniTokenMapper, tokenMapper)
	return routes
}
