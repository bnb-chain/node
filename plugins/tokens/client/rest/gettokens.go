package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/wire"
)

const maxTokensLimit = int64(1000)
const defaultTokensLimit = int64(100)
const defaultTokensOffset = int64(0)

func listAllTokens(ctx context.CLIContext, cdc *wire.Codec, offset int64, limit int64) ([]types.Token, error) {
	bz, err := ctx.Query(fmt.Sprintf("tokens/list/%d/%d", offset, limit), nil)
	if err != nil {
		return nil, err
	}
	tokens := make([]types.Token, 0)
	err = cdc.UnmarshalBinaryLengthPrefixed(bz, &tokens)
	return tokens, nil
}

// GetTokensReqHandler creates an http request handler to get the list of tokens in the token mapper
func GetTokensReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	type params struct {
		limit  int64
		offset int64
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
		limit := defaultTokensLimit
		if limitStr != "" && len(limitStr) < 100 {
			parsed, err := strconv.ParseInt(limitStr, 10, 64)
			if err != nil {
				throw(w, http.StatusExpectationFailed, errors.New("invalid limit"))
				return
			}
			limit = parsed
		}

		// validate and use offset param
		offset := defaultTokensOffset
		if offsetStr != "" && len(offsetStr) < 100 {
			parsed, err := strconv.ParseInt(offsetStr, 10, 64)
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

		// apply max tokens limit
		if params.limit > maxTokensLimit {
			params.limit = maxTokensLimit
		}

		tokens, err := listAllTokens(ctx, cdc, params.offset, params.limit)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		output, err := cdc.MarshalJSON(tokens)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
