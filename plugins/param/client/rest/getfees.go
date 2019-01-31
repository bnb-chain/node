package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/binance-chain/node/plugins/param"
	"github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"
)

func GetFeesParamHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {

	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {

		bz, err := ctx.Query(fmt.Sprintf("%s/fees", param.AbciQueryPrefix), nil)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
		var fees []types.FeeParam
		err = cdc.UnmarshalBinaryLengthPrefixed(bz, &fees)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		output, err := json.Marshal(fees)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
