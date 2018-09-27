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

// DepthReqHandler creates an http request handler to show market depth data
func DepthReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	allowedLimitsA := strings.Split(allowedLimits, ",")

	type params struct {
		symbol string
		limit  int
	}

	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Header().Set("Content-Type", "text/plain")
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

		limit, _ := strconv.Atoi(defaultLimit)
		if len(limitStrOk) > 0 {
			var err error
			limit, err = strconv.Atoi(limitStrOk)
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

		levels := make(chan []store.OrderBookLevel, 1)
		height := make(chan int64, 1)
		errs := make(chan error, 2)

		// ROUTINE 1/2 - query OB levels
		obWorker := func(levels chan []store.OrderBookLevel, errs chan error) {
			lvls, err := store.GetOrderBookLevels(cdc, ctx, params.symbol)
			if err != nil {
				errs <- err
				return
			}
			levels <- lvls
		}

		// ROUTINE 2/2 - query block height
		heightWorker := func(height chan int64, errs chan error) {
			node, err := ctx.GetNode()
			if err != nil {
				errs <- err
				return
			}
			status, err := node.Status()
			if err != nil {
				errs <- err
				return
			}
			height <- status.SyncInfo.LatestBlockHeight
		}

		go obWorker(levels, errs)
		go heightWorker(height, errs)

		var ob store.OrderBook
		for i := 0; i < 2; i++ {
			select {
			case err := <-errs:
				throw(w, http.StatusInternalServerError, err)
				return
			case hght := <-height:
				ob.Height = hght
			case lvls := <-levels:
				ob.Levels = lvls
			}
		}

		w.Header().Set("Content-Type", responseType)

		err = rutils.StreamDepthResponse(w, ob, limit)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
	}
}
