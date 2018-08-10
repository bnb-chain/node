package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

// DepthReqHandler creates an http request handler to send coins to a address
func DepthReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	type params struct {
		pair string
	}
	type response struct {
		Pair   string        `json:"pair"`
		Orders []store.Order `json:"orders"`
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

		book, err := store.GetOrderBook(cdc, ctx, params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		resp := response{
			Pair:   vars["pair"],
			Orders: *book,
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Write(output)
	}
}
