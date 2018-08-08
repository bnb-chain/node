package api

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"

	hnd "github.com/BiJie/BinanceChain/plugins/api/handlers"
	dexapi "github.com/BiJie/BinanceChain/plugins/dex/client/rest"
	tksapi "github.com/BiJie/BinanceChain/plugins/tokens/client/rest"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/BiJie/BinanceChain/wire"
)

func (s *server) handleVersion() http.HandlerFunc {
	return hnd.CLIVersionRequestHandler
}

func (s *server) handleNodeVersion() http.HandlerFunc {
	return hnd.NodeVersionRequestHandler(s.ctx)
}

func (s *server) handleDexDepthRequest(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	return dexapi.DepthRequestHandler(cdc, ctx)
}

func (s *server) handleBalancesRequest(cdc *wire.Codec, ctx context.CoreContext, tokens tkstore.Mapper) http.HandlerFunc {
	return tksapi.BalancesRequestHandler(cdc, ctx, tokens)
}

func (s *server) handleBalanceRequest(cdc *wire.Codec, ctx context.CoreContext, tokens tkstore.Mapper) http.HandlerFunc {
	return tksapi.BalanceRequestHandler(cdc, ctx, tokens)
}
