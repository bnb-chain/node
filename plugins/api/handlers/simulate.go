package handlers

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cctx "github.com/BiJie/BinanceChain/common/client/context"
	"github.com/BiJie/BinanceChain/wire"
)

// SimulateReqHandler simulates the execution of a single transaction, given its binary form
func SimulateReqHandler(cdc *wire.Codec, ctx context.CoreContext) http.HandlerFunc {
	type response sdk.Result
	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, message string) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(message))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)

		if err != nil {
			errMsg := fmt.Sprintf("Malformed request body. Error: %s", err.Error())
			throw(w, http.StatusExpectationFailed, errMsg)
			return
		}

		bz := make([]byte, len(body)/2)
		_, err = hex.Decode(bz, body)
		if err != nil {
			errMsg := fmt.Sprintf("Couldn't decode hex body. Error: %s", err.Error())
			throw(w, http.StatusExpectationFailed, errMsg)
			return
		}

		res, err := cctx.QueryWithData(ctx, "/app/simulate", bz)
		if err != nil {
			errMsg := fmt.Sprintf("Couldn't simulate transaction. Error: %s", err.Error())
			throw(w, http.StatusExpectationFailed, errMsg)
			return
		}

		// expect abci query result to be `sdk.Result`
		var resp response
		err = cdc.UnmarshalBinary(res, &resp)
		if err != nil {
			errMsg := fmt.Sprintf("Couldn't unmarshal. Error: %s. Response: %s", err.Error(), res)
			throw(w, http.StatusInternalServerError, errMsg)
			return
		}

		// re-marshal to json
		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			errMsg := fmt.Sprintf("Couldn't marshal. Error: %s", err.Error())
			throw(w, http.StatusInternalServerError, errMsg)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
