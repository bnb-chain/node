package rest

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/gorilla/mux"

	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/wire"
)

// QuerySwapReqHandler creates an http request handler to get AtomicSwap record by randomNumberHash
func QuerySwapReqHandler(
	cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		randomNumberHashStr := vars["randomNumberHash"]
		if !strings.HasPrefix(randomNumberHashStr, "0x") {
			throw(w, http.StatusBadRequest, fmt.Errorf("randomNumberHash must prefix with 0x"))
			return
		}
		randomNumberHash, err := hex.DecodeString(randomNumberHashStr[2:])
		if err != nil {
			throw(w, http.StatusBadRequest, err)
			return
		}
		params := swap.QuerySwapByHashParams{
			RandomNumberHash: randomNumberHash,
		}

		bz, err := cdc.MarshalJSON(params)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		output, err := ctx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapByHashParams{}), bz)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
