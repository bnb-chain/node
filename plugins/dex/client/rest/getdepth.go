package rest

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"

	rutils "github.com/BiJie/BinanceChain/plugins/dex/client/rest/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

const allowedLimits = "5,10,20,50,100"
const defaultLimit = "100"
const maxUint = ^uint(0)

// DepthReqHandler creates an http request handler to send coins to a address
func DepthReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	allowedLimitsA := strings.Split(allowedLimits, ",")
	type params struct {
		symbol string
		limit  int
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// collect params
		limitStr := r.FormValue("limit")

		// validate limit param
		limitStrOk := defaultLimit
		for _, lmt := range allowedLimitsA {
			if lmt == limitStr {
				limitStrOk = limitStr
				break
			}
		}

		limit := int(maxUint >> 1)
		if len(limitStrOk) > 0 {
			var err error
			limit, err = strconv.Atoi(limitStrOk)
			if err != nil {
				throw(w, http.StatusExpectationFailed, errors.New("invalid limit"))
				return
			}
		}

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

		table, err := store.GetOrderBook(cdc, ctx, params.symbol)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		err = rutils.StreamDepthResponse(w, table, limit)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
	}
}
