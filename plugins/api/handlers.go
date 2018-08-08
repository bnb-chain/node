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

func (s *server) handleVersionReq() http.HandlerFunc {
	return hnd.CLIVersionReqHandler
}

func (s *server) handleNodeVersionReq() http.HandlerFunc {
	return hnd.NodeVersionReqHandler(s.ctx)
}

func (s *server) handleDexDepthReq(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	return dexapi.DepthReqHandler(cdc, ctx)
}

func (s *server) handleBalancesReq(cdc *wire.Codec, ctx context.CoreContext, tokens tkstore.Mapper) http.HandlerFunc {
	return tksapi.BalancesReqHandler(cdc, ctx, tokens)
}

func (s *server) handleBalanceReq(cdc *wire.Codec, ctx context.CoreContext, tokens tkstore.Mapper) http.HandlerFunc {
	return tksapi.BalanceReqHandler(cdc, ctx, tokens)
}
