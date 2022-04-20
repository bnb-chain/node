package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"

	"github.com/bnb-chain/node/plugins/tokens/timelock"
	"github.com/bnb-chain/node/wire"
)

func getTimeLocks(ctx context.CLIContext, cdc *wire.Codec, address sdk.AccAddress) ([]timelock.TimeLockRecord, error) {
	params := timelock.QueryTimeLocksParams{
		Account: address,
	}

	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return nil, err
	}

	bz, err = ctx.QueryWithData(fmt.Sprintf("custom/%s/%s", timelock.MsgRoute, timelock.QueryTimeLocks), bz)
	if err != nil {
		return nil, err
	}

	var records []timelock.TimeLockRecord
	err = cdc.UnmarshalJSON(bz, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func GetTimeLocksReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
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
		var addressStr string

		if _, ok := vars["address"]; ok {
			addressStr = vars["address"]
		} else {
			throw(w, http.StatusBadRequest, fmt.Errorf("miss request parameter `address`"))
			return
		}

		address, err := sdk.AccAddressFromBech32(addressStr)
		if err != nil {
			throw(w, http.StatusBadRequest, fmt.Errorf("invalid address, address=%s", addressStr))
			return
		}

		if len(address) != sdk.AddrLen {
			throw(w, http.StatusBadRequest, fmt.Errorf("address length should be %d", sdk.AddrLen))
			return
		}

		records, err := getTimeLocks(ctx, cdc, address)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		if records == nil {
			throw(w, http.StatusInternalServerError, errors.New("no time locks found"))
			return
		}

		// no need to use cdc here because we do not want amino to inject a type attribute
		output, err := json.Marshal(records)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
