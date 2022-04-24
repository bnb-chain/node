package rest

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"

	rutils "github.com/bnb-chain/node/plugins/dex/client/rest/utils"
	"github.com/bnb-chain/node/plugins/dex/store"
	"github.com/bnb-chain/node/wire"
)

var allowedLimits = [7]int{5, 10, 20, 50, 100, 500, 1000}

// DepthReqHandler creates an http request handler to show market depth data
func DepthReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {

	type params struct {
		symbol string
		limit  int
	}

	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(err.Error()))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.FormValue("limit")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			throw(w, http.StatusExpectationFailed, errors.New("invalid limit, supported limits: [5,10,20,50,100,500,1000]"))
			return
		}

		// validate limit param
		limitOk := -1
		for _, lmt := range allowedLimits {
			if lmt == limit {
				limitOk = lmt
				break
			}
		}

		if limitOk == -1 {
			throw(w, http.StatusExpectationFailed, errors.New("invalid limit, supported limits: [5,10,20,50,100,500,1000]"))
			return
		}

		// collect params
		params := params{
			symbol: r.FormValue("symbol"),
			limit:  limit,
		}

		// validate pair
		err = store.ValidatePairSymbol(params.symbol)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		// query order book (includes block height)
		ob, err := store.GetOrderBook(cdc, ctx, params.symbol, params.limit)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)

		err = rutils.StreamDepthResponse(w, ob, limit)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
	}
}
