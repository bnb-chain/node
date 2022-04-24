package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"

	"github.com/bnb-chain/node/plugins/tokens/timelock"
	"github.com/bnb-chain/node/wire"
)

func getTimeLock(ctx context.CLIContext, cdc *wire.Codec, address sdk.AccAddress, id int64) (timelock.TimeLockRecord, error) {
	params := timelock.QueryTimeLockParams{
		Account: address,
		Id:      id,
	}

	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return timelock.TimeLockRecord{}, err
	}

	bz, err = ctx.QueryWithData(fmt.Sprintf("custom/%s/%s", timelock.MsgRoute, timelock.QueryTimeLock), bz)
	if err != nil {
		return timelock.TimeLockRecord{}, err
	}

	var record timelock.TimeLockRecord
	err = cdc.UnmarshalJSON(bz, &record)
	if err != nil {
		return timelock.TimeLockRecord{}, err
	}

	return record, nil
}

func GetTimeLockReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(err.Error()))
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

		var idStr string
		if _, ok := vars["id"]; ok {
			idStr = vars["id"]
		} else {
			throw(w, http.StatusBadRequest, fmt.Errorf("miss request parameter `id`"))
			return
		}

		id, _ := strconv.ParseInt(idStr, 10, 0)
		if id < timelock.InitialRecordId {
			throw(w, http.StatusBadRequest, fmt.Errorf("id(%d) should not less than %d", id, timelock.InitialRecordId))
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

		record, err := getTimeLock(ctx, cdc, address, id)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		// no need to use cdc here because we do not want amino to inject a type attribute
		output, err := json.Marshal(record)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(output)
	}
}
