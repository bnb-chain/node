package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tendermint/go-amino"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/x/paramHub"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

func GetFeesParamHandler(cdc *amino.Codec, ctx context.CLIContext) http.HandlerFunc {

	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {

		bz, err := ctx.Query(fmt.Sprintf("%s/fees", paramHub.AbciQueryPrefix), nil)
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
		formats, exist := r.URL.Query()["format"]
		format := types.JSONFORMAT
		if exist {
			if len(formats) < 1 {
				throw(w, http.StatusBadRequest, errors.New(fmt.Sprintf("Format parameter is invalid")))
				return
			}
			format = formats[0]
			if format != types.JSONFORMAT && format != types.AMINOFORMAT {
				throw(w, http.StatusBadRequest, errors.New(fmt.Sprintf("Format %s is not supported, options [%s, %s]", format, types.JSONFORMAT, types.AMINOFORMAT)))
				return
			}
		}
		var output []byte
		if format == types.JSONFORMAT {
			output, err = json.Marshal(fees)
		} else if format == types.AMINOFORMAT {
			output, err = cdc.MarshalJSON(fees)
		}
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
