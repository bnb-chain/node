package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

// DepthReqHandler creates an http request handler to send coins to a address
func DepthReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	type params struct {
		pair string
	}
	type response struct {
		Pair   string        `json:"pair"`
		Orders []store.Order `json:"orders"`
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	write := func(w http.ResponseWriter, data string) error {
		if _, err := w.Write([]byte(data)); err != nil {
			throw(w, http.StatusInternalServerError, err)
			return err
		}
		return nil
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		// collect params
		params := params{
			pair: vars["pair"],
		}

		// validate pair
		err := store.ValidatePairSymbol(params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		table, err := store.GetOrderBookRaw(cdc, ctx, params.pair)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		if err = write(w, "{\"asks\":["); err != nil {
			return
		}

		// pass 1 - asks
		i := 0
		for _, o := range *table {
			if i > 0 {
				if err = write(w, ","); err != nil {
					return
				}
			}
			// [PRICE, QTY]
			if err = write(w, fmt.Sprintf("[\"%s\",\"%s\"]", utils.Fixed8(o[1]), utils.Fixed8(o[0]))); err != nil {
				return
			}
			i++
		}

		// pass 2 - bids
		if err = write(w, "],\"bids\":["); err != nil {
			return
		}
		i = 0
		for _, o := range *table {
			if i > 0 {
				if err = write(w, ","); err != nil {
					return
				}
			}
			// [PRICE, QTY]
			if err = write(w, fmt.Sprintf("[\"%s\",\"%s\"]", utils.Fixed8(o[2]), utils.Fixed8(o[3]))); err != nil {
				return
			}
			i++
		}

		// end streamed json
		if err = write(w, "]}"); err != nil {
			return
		}
	}
}
