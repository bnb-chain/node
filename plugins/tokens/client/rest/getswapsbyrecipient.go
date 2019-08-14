package rest

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/wire"
)

// QuerySwapsByRecipientReqHandler creates an http request handler to
func QuerySwapsByRecipientReqHandler(
	cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		recipientAddr, err := sdk.AccAddressFromBech32(vars["recipientAddr"])
		if err != nil {
			throw(w, http.StatusBadRequest, err)
			return
		}
		swapStatus := swap.NewSwapStatusFromString(vars["swapStatus"])
		limitStr := r.FormValue("limit")
		offsetStr := r.FormValue("offset")

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			throw(w, http.StatusBadRequest, fmt.Errorf("invalid limit"))
			return
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			throw(w, http.StatusBadRequest, fmt.Errorf("invalid offset"))
			return
		}

		params := swap.QuerySwapByRecipientParams{
			Recipient: recipientAddr,
			Status:    swapStatus,
			Limit:     int64(limit),
			Offset:    int64(offset),
		}

		bz, err := cdc.MarshalJSON(params)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		output, err := ctx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapRecipient), bz)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}