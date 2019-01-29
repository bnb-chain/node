package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

const maxPairsLimit = 1000
const defaultPairsLimit = 100
const defaultPairsOffset = 0

func listAllTradingPairs(ctx context.CLIContext, cdc *wire.Codec, offset int, limit int) ([]types.TradingPair, error) {
	bz, err := ctx.Query(fmt.Sprintf("dex/pairs/%d/%d", offset, limit), nil)
	if err != nil {
		return nil, err
	}
	pairs := make([]types.TradingPair, 0)
	err = cdc.UnmarshalBinaryLengthPrefixed(bz, &pairs)
	return pairs, nil
}

// GetPairsReqHandler creates an http request handler to list
func GetPairsReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	type params struct {
		limit  int
		offset int
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
		offsetStr := r.FormValue("offset")

		// validate and use limit param
		limit := defaultPairsLimit
		if limitStr != "" && len(limitStr) < 100 {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				throw(w, http.StatusExpectationFailed, errors.New("invalid limit"))
				return
			}
			limit = parsed
		}

		// validate and use offset param
		offset := defaultPairsOffset
		if offsetStr != "" && len(offsetStr) < 100 {
			parsed, err := strconv.Atoi(offsetStr)
			if err != nil {
				throw(w, http.StatusExpectationFailed, errors.New("invalid offset"))
				return
			}
			offset = parsed
		}

		// collect params
		params := params{
			limit:  limit,
			offset: offset,
		}

		// apply max pairs limit
		if params.limit > maxPairsLimit {
			params.limit = maxPairsLimit
		}

		pairs, err := listAllTradingPairs(ctx, cdc, params.offset, params.limit)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		output, err := cdc.MarshalJSON(pairs)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
