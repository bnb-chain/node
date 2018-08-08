package rest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

// DepthRequestHandler creates an http request handler to send coins to a address
func DepthRequestHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	type params struct {
		pair string
	}
	type response struct {
		Pair   string  `json:"pair"`
		Orders []order `json:"orders"`
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		// collect params
		params := params{
			pair: vars["pair"],
		}

		// validate pair
		err := validatePairSymbol(params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		book, err := getOrderBook(cdc, ctx, params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		resp := response{
			Pair:   vars["pair"],
			Orders: *book,
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Write(output)
	}
}

func validatePairSymbol(symbol string) error {
	tokenSymbols := strings.Split(symbol, "_")
	if len(tokenSymbols) != 2 {
		return errors.New("Invalid symbol")
	}

	for _, tokenSymbol := range tokenSymbols {
		err := types.ValidateSymbol(tokenSymbol)
		if err != nil {
			return err
		}
	}

	return nil
}
