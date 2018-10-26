package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

const maxTokensLimit = 1000
const defaultTokensLimit = 100
const defaultTokensOffset = 0

func listAllTokens(ctx context.CLIContext, cdc *wire.Codec, offset int, limit int) ([]types.Token, error) {
	bz, err := ctx.Query(fmt.Sprintf("tokens/list/%d/%d", offset, limit), nil)
	if err != nil {
		return nil, err
	}
	tokens := make([]types.Token, 0)
	err = cdc.UnmarshalBinary(bz, &tokens)
	return tokens, nil
}

// GetTokensReqHandler creates an http request handler to get the list of tokens in the token mapper
func GetTokensReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	type params struct {
		limit  int
		offset int
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
		offsetStr := r.FormValue("offset")

		// validate and use limit param
		limit := defaultTokensLimit
		if limitStr != "" && len(limitStr) < 100 {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				throw(w, http.StatusExpectationFailed, errors.New("invalid limit"))
				return
			}
			limit = parsed
		}

		// validate and use offset param
		offset := defaultTokensOffset
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

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", responseType)
		w.Write(output)
	}
}
