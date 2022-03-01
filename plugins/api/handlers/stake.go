package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/gorilla/mux"
	"github.com/tendermint/tendermint/crypto"
	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/bnb-chain/node/wire"
)

// ValidatorQueryReqHandler queries the whole validator set
func ValidatorQueryReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {

	type ValidatorOutput struct {
		AccountAddr        sdk.AccAddress    `json:"account_address"`
		OperatorAddr       sdk.ValAddress    `json:"operator_address"`
		ConsPubKey         crypto.PubKey     `json:"consensus_pubkey"`
		ConsAddr           cmn.HexBytes      `json:"consensus_address"`
		Jailed             bool              `json:"jailed"`
		Status             string            `json:"status"`
		Tokens             sdk.Dec           `json:"tokens"`
		Power              int64             `json:"power"`
		DelegatorShares    sdk.Dec           `json:"delegator_shares"`
		Description        stake.Description `json:"description"`
		BondHeight         int64             `json:"bond_height"`
		BondIntraTxCounter int16             `json:"bond_intra_tx_counter"`
		UnbondingHeight    int64             `json:"unbonding_height"`
		UnbondingMinTime   time.Time         `json:"unbonding_time"`
		Commission         stake.Commission  `json:"commission"`
	}

	convertToValidatorOutputs := func(validators []stake.Validator) (validatorOutputs []ValidatorOutput) {
		for _, val := range validators {
			validatorOutputs = append(validatorOutputs, ValidatorOutput{
				AccountAddr:        val.FeeAddr,
				OperatorAddr:       val.OperatorAddr,
				ConsPubKey:         val.ConsPubKey,
				ConsAddr:           val.ConsPubKey.Address(),
				Jailed:             val.Jailed,
				Status:             sdk.BondStatusToString(val.Status),
				Tokens:             val.Tokens,
				Power:              val.GetPower().RawInt(),
				DelegatorShares:    val.DelegatorShares,
				Description:        val.Description,
				BondHeight:         val.BondHeight,
				BondIntraTxCounter: val.BondIntraTxCounter,
				UnbondingHeight:    val.UnbondingHeight,
				UnbondingMinTime:   val.UnbondingMinTime,
				Commission:         val.Commission,
			})
		}
		return
	}

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
		json.NewEncoder(w).Encode(convertToValidatorOutputs(validators))
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
