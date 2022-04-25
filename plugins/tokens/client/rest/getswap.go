package rest

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/gorilla/mux"

	"github.com/bnb-chain/node/plugins/tokens/swap"
	"github.com/bnb-chain/node/wire"
)

// QuerySwapReqHandler creates an http request handler to query an AtomicSwap record by swapID
func QuerySwapReqHandler(
	cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(err.Error()))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		swapID, err := hex.DecodeString(vars["swapID"])
		if err != nil {
			throw(w, http.StatusBadRequest, err)
			return
		}
		if len(swapID) != swap.SwapIDLength {
			throw(w, http.StatusBadRequest, fmt.Errorf("length of swapID should be %d", swap.SwapIDLength))
			return
		}

		params := swap.QuerySwapByID{
			SwapID: swapID,
		}

		bz, err := cdc.MarshalJSON(params)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		output, err := ctx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapID), bz)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
		if output == nil {
			throw(w, http.StatusNotFound, fmt.Errorf("no match swapID"))
			return
		}
		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(output)
	}
}
