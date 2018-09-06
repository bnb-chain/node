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

const defaultPairsLimit = 100
const defaultPairsOffset = 0

func listAllTradingPairs(ctx context.CoreContext, cdc *wire.Codec, offset int, limit int) ([]types.TradingPair, error) {
	bz, err := ctx.Query(fmt.Sprintf("dex/pairs/%d/%d", offset, limit))
	if err != nil {
		return nil, err
	}
	pairs := make([]types.TradingPair, 0)
	err = cdc.UnmarshalBinary(bz, &pairs)
	return pairs, nil
}

// GetPairsReqHandler creates an http request handler to list
func GetPairsReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	type params struct {
		limit  int
		offset int
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.FormValue("limit")
		offsetStr := r.FormValue("offset")

		// validate and use limit param
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			if len(limitStr) > 0 {
				throw(w, http.StatusExpectationFailed, errors.New("invalid limit"))
				return
			} else {
				limit = defaultPairsLimit
			}
		}

		// validate and use offset param
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			if len(offsetStr) > 0 {
				throw(w, http.StatusExpectationFailed, errors.New("invalid offset"))
				return
			} else {
				offset = defaultPairsOffset
			}
		}

		// collect params
		params := params{
			limit:  limit,
			offset: offset,
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

		w.Write(output)
	}
}
