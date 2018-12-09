package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

func getTokenInfo(ctx context.CLIContext, cdc *wire.Codec, symbol string) (*types.Token, error) {
	bz, err := ctx.Query(fmt.Sprintf("tokens/info/%s", symbol), nil)
	if err != nil {
		return nil, err
	}
	var token types.Token
	err = cdc.UnmarshalBinary(bz, &token)
	return &token, nil
}

// GetTokenReqHandler creates an http request handler to get info for an individual token
func GetTokenReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	type params struct {
		symbol string
	}

	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// validate and use symbol param
		vars := mux.Vars(r)
		var params params
		if _, ok := vars["symbol"]; ok {
			params.symbol = vars["symbol"]
		} else {
			throw(w, http.StatusExpectationFailed, errors.New("invalid symbol"))
			return
		}

		if len(params.symbol) == 0 || len(params.symbol) > 100 {
			throw(w, http.StatusExpectationFailed, errors.New("invalid symbol"))
			return
		}

		token, err := getTokenInfo(ctx, cdc, params.symbol)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
		if token == nil {
			throw(w, http.StatusInternalServerError, errors.New("token is nil"))
			return
		}

		// no need to use cdc here because we do not want amino to inject a type attribute
		output, err := json.Marshal(token)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
