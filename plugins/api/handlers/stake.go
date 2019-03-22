package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/binance-chain/node/wire"
	"github.com/gorilla/mux"
)

// ValidatorQueryReqHandler queries the whole validator set
func ValidatorQueryReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {

	throw := func(w http.ResponseWriter, status int, message string) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(message))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {

		res, err := ctx.QueryWithData("custom/stake/validators", nil)
		if err != nil {
			throw(w, http.StatusInternalServerError, err.Error())
			return
		}

		var validators []stake.Validator
		err = cdc.UnmarshalJSON(res, &validators)
		if err != nil {
			throw(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(validators)
	}
}

// DelegatorUnbondindDelegationsQueryReqHandler queries all unbonding delegations of the given delegator
func DelegatorUnbondindDelegationsQueryReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {

	throw := func(w http.ResponseWriter, status int, message string) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(message))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bech32delegator := vars["delegatorAddr"]
		delegatorAddr, err := sdk.AccAddressFromBech32(bech32delegator)
		if err != nil {
			throw(w, http.StatusBadRequest, err.Error())
			return
		}

		params := stake.QueryDelegatorParams{
			DelegatorAddr: delegatorAddr,
		}

		bz, err := cdc.MarshalJSON(params)
		if err != nil {
			throw(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := ctx.QueryWithData("custom/stake/delegatorUnbondingDelegations", bz)
		if err != nil {
			throw(w, http.StatusInternalServerError, err.Error())
			return
		}

		var unbondingDelegations []stake.UnbondingDelegation
		err = cdc.UnmarshalJSON(res, &unbondingDelegations)
		if err != nil {
			throw(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(unbondingDelegations)
	}
}
