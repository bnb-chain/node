package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"
	cmm "github.com/tendermint/tendermint/libs/common"

	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/wire"
)

// QuerySwapIDsByCreatorReqHandler creates an http request handler to query swapID list by creator address
func QuerySwapIDsByCreatorReqHandler(
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

		creatorAddr, err := sdk.AccAddressFromBech32(vars["creatorAddr"])
		if err != nil {
			throw(w, http.StatusBadRequest, err)
			return
		}

		limitStr := r.FormValue("limit")
		offsetStr := r.FormValue("offset")

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			throw(w, http.StatusBadRequest, fmt.Errorf("invalid limit"))
			return
		}
		if limit <= 0 || limit > 100 {
			throw(w, http.StatusBadRequest, fmt.Errorf("limit should be in (0, 100]"))
			return
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			throw(w, http.StatusBadRequest, fmt.Errorf("invalid offset"))
			return
		}
		if offset < 0 {
			throw(w, http.StatusBadRequest, fmt.Errorf("offset must be positive"))
			return
		}

		params := swap.QuerySwapByCreatorParams{
			Creator: creatorAddr,
			Limit:   int64(limit),
			Offset:  int64(offset),
		}

		paramsBytes, err := cdc.MarshalJSON(params)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		bz, err := ctx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapCreator), paramsBytes)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		var swapIDs []cmm.HexBytes
		err = cdc.UnmarshalJSON(bz, &swapIDs)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		if len(swapIDs) == 0 {
			throw(w, http.StatusNotFound, fmt.Errorf("no match swapID"))
			return
		}

		output, err := json.MarshalIndent(swapIDs, "", "  ")
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
