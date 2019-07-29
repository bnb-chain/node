package rest

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/wire"
)

// QuerySwapReqHandler creates an http request handler to get AtomicSwap record by randomeNumberHash
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
		hashKey := swap.GetSwapHashKey(randomNumberHash)

		res, err := ctx.QueryStore(hashKey, common.AtomicSwapStoreName)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		if res == nil {
			throw(w, http.StatusBadRequest, fmt.Errorf("no matched swap record"))
			return
		}

		atomicSwap := swap.DecodeAtomicSwap(cdc, res)
		output, err := cdc.MarshalJSON(atomicSwap)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
