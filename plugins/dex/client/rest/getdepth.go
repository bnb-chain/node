package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"

	rutils "github.com/BiJie/BinanceChain/plugins/dex/client/rest/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

// DepthReqHandler creates an http request handler to send coins to a address
func DepthReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	type params struct {
		pair string
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		// collect params
		params := params{
			pair: vars["pair"],
		}

		// validate pair
		err := store.ValidatePairSymbol(params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		table, err := store.GetOrderBookRaw(cdc, ctx, params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		err = rutils.StreamDepthResponse(w, table)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
	}
}
