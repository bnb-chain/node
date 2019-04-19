package rest

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"

	rutils "github.com/binance-chain/node/plugins/dex/client/rest/utils"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/wire"
)

const allowedLimits = "5,10,20,50,100"
const defaultLimit = "100"

// DepthReqHandler creates an http request handler to show market depth data
func DepthReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	allowedLimitsA := strings.Split(allowedLimits, ",")

	type params struct {
		symbol string
		limit  int64
	}

	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.FormValue("limit")

		// validate limit param
		limitStrOk := defaultLimit
		for _, lmt := range allowedLimitsA {
			if lmt == limitStr {
				limitStrOk = limitStr
				break
			}
		}

		limit, _ := strconv.ParseInt(defaultLimit, 10, 64)
		if len(limitStrOk) > 0 {
			var err error
			limit, err = strconv.ParseInt(limitStrOk, 10, 64)
			if err != nil {
				throw(w, http.StatusExpectationFailed, errors.New("invalid limit"))
				return
			}
		}

		// collect params
		params := params{
			symbol: r.FormValue("symbol"),
			limit:  limit,
		}

		// validate pair
		err := store.ValidatePairSymbol(params.symbol)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		// query order book (includes block height)
		ob, err := store.GetOrderBook(cdc, ctx, params.symbol)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)

		err = rutils.StreamDepthResponse(w, ob, int(limit))
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
	}
}
