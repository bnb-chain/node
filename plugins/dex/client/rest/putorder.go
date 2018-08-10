package rest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

// PutOrderReqHandler creates an http request handler to send coins to a address
func PutOrderReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	type formParams struct {
		address string
		pair    string
		orderID string
		price   string
		qty     string
		tif     string
	}
	type response struct {
		OK      bool   `json:"ok"`
		OrderID string `json:"order_id"`
	}
	validateFormParams := func(params formParams) bool {
		// TODO: there might be a better way to do this
		if strings.TrimSpace(params.address) == "" {
			return false
		}
		if strings.TrimSpace(params.pair) == "" {
			return false
		}
		if strings.TrimSpace(params.orderID) == "" {
			return false
		}
		if strings.TrimSpace(params.price) == "" {
			return false
		}
		if strings.TrimSpace(params.qty) == "" {
			return false
		}
		if strings.TrimSpace(params.tif) == "" {
			return false
		}
		return true
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// collect params
		params := formParams{
			address: r.FormValue("address"),
			pair:    r.FormValue("pair"),
			orderID: r.FormValue("order_id"),
			price:   r.FormValue("price"),
			qty:     r.FormValue("qty"),
			tif:     r.FormValue("tif"),
		}

		if !validateFormParams(params) {
			throw(w, http.StatusExpectationFailed, errors.New("invalid arguments"))
			return
		}

		// validate pair
		err := store.ValidatePairSymbol(params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		resp := response{
			OK:      true,
			OrderID: "xyz",
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Write(output)
	}
}
