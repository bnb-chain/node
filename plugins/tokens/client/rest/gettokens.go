package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/wire"
)

const maxTokensLimit = 1000
const defaultTokensLimit = 100
const defaultTokensOffset = 0

func listAllTokens(ctx context.CLIContext, cdc *wire.Codec, offset int, limit int, showZeroSupplyTokens bool, isMini bool) (interface{}, error) {
	var abciPrefix string
	if isMini {
		abciPrefix = "mini-tokens"
	} else {
		abciPrefix = "tokens"
	}
	bz, err := ctx.Query(fmt.Sprintf("%s/list/%d/%d/%s", abciPrefix, offset, limit, strconv.FormatBool(showZeroSupplyTokens)), nil)
	if err != nil {
		return nil, err
	}

	tokens := make([]types.IToken, 0)
	err = cdc.UnmarshalBinaryLengthPrefixed(bz, &tokens)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// GetTokensReqHandler creates an http request handler to get the list of tokens in the token mapper
func GetTokensReqHandler(cdc *wire.Codec, ctx context.CLIContext, isMini bool) http.HandlerFunc {
	type params struct {
		limit                int
		offset               int
		showZeroSupplyTokens bool
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
		showZeroSupplyTokensStr := r.FormValue("showZeroSupplyTokens")

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

		showZeroSupplyTokens := false
		if strings.ToLower(showZeroSupplyTokensStr) == "true" {
			showZeroSupplyTokens = true
		}

		// collect params
		params := params{
			limit:                limit,
			offset:               offset,
			showZeroSupplyTokens: showZeroSupplyTokens,
		}

		// apply max tokens limit
		if params.limit > maxTokensLimit {
			params.limit = maxTokensLimit
		}

		tokens, err := listAllTokens(ctx, cdc, params.offset, params.limit, params.showZeroSupplyTokens, isMini)
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
