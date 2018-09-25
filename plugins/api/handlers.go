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

// middleware (limits, parsing, etc)

func (s *server) limitReqSize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// reject suspiciously large form posts
		if r.ContentLength > s.maxPostSize {
			http.Error(w, "request too large", http.StatusExpectationFailed)
			return
		}
		// parse request body as multipart/form-data with maxPostSize in mind
		r.Body = http.MaxBytesReader(w, r.Body, s.maxPostSize)
		next(w, r)
	}
}

func (s *server) withUrlEncForm(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		next(w, r)
	}
}

func (s *server) withMultipartForm(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(1024)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		next(w, r)
	}
}

// -----

func (s *server) handleVersionReq() http.HandlerFunc {
	return hnd.CLIVersionReqHandler
}

func (s *server) handleNodeVersionReq() http.HandlerFunc {
	return hnd.NodeVersionReqHandler(s.ctx)
}

func (s *server) handlePairsReq(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	return dexapi.GetPairsReqHandler(cdc, ctx)
}

func (s *server) handleDexDepthReq(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	return dexapi.DepthReqHandler(cdc, ctx)
}

func (s *server) handleDexOrderReq(cdc *wire.Codec, ctx context.CoreContext, accStoreName string) http.HandlerFunc {
	h := dexapi.PutOrderReqHandler(cdc, ctx, accStoreName)
	return s.withUrlEncForm(s.limitReqSize(h))
}

func (s *server) handleDexOpenOrdersReq(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	return dexapi.OpenOrdersReqHandler(cdc, ctx)
}

func (s *server) handleTokenReq(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	return tksapi.GetTokenReqHandler(cdc, ctx)
}

func (s *server) handleTokensReq(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	return tksapi.GetTokensReqHandler(cdc, ctx)
}

func (s *server) handleBalancesReq(cdc *wire.Codec, ctx context.CoreContext, tokens tkstore.Mapper) http.HandlerFunc {
	return tksapi.BalancesReqHandler(cdc, ctx, tokens)
}

func (s *server) handleBalanceReq(cdc *wire.Codec, ctx context.CoreContext, tokens tkstore.Mapper) http.HandlerFunc {
	return tksapi.BalanceReqHandler(cdc, ctx, tokens)
}
