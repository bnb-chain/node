package rest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/bsc"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/slashing"
)

func registerTxRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec, kb keys.Keybase) {
	r.HandleFunc(
		"/slashing/validators/{validatorAddr}/unjail",
		unjailRequestHandlerFn(cdc, kb, cliCtx),
	).Methods("POST")

	r.HandleFunc(
		"/slashing/bsc/evidence/submit",
		bscEvidenceSubmitRequestHandlerFn(cdc, kb, cliCtx),
	).Methods("POST")
}

// Unjail TX body
type UnjailReq struct {
	BaseReq utils.BaseReq `json:"base_req"`
}

func unjailRequestHandlerFn(cdc *codec.Codec, kb keys.Keybase, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		bech32validator := vars["validatorAddr"]

		var req UnjailReq
		err := utils.ReadRESTReq(w, r, cdc, &req)
		if err != nil {
			return
		}

		baseReq := req.BaseReq.Sanitize()
		if !baseReq.ValidateBasic(w) {
			return
		}

		info, err := kb.Get(baseReq.Name)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}

		valAddr, err := sdk.ValAddressFromBech32(bech32validator)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		if !bytes.Equal(info.GetPubKey().Address(), valAddr) {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, "must use own validator address")
			return
		}

		msg := slashing.NewMsgUnjail(valAddr)
		utils.CompleteAndBroadcastTxREST(w, r, cliCtx, baseReq, []sdk.Msg{msg}, cdc)
	}
}

type EvidenceSubmitReq struct {
	BaseReq   utils.BaseReq `json:"base_req"`
	Submitter string        `json:"submitter"` // in bech 32
	Headers   []bsc.Header  `json:"headers"`
}

func bscEvidenceSubmitRequestHandlerFn(cdc *codec.Codec, kb keys.Keybase, cliCtx context.CLIContext) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		var req EvidenceSubmitReq
		err = json.Unmarshal(body, &req)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		baseReq := req.BaseReq.Sanitize()
		if !baseReq.ValidateBasic(w) {
			return
		}

		info, err := kb.Get(baseReq.Name)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}

		submitter, err := sdk.AccAddressFromBech32(req.Submitter)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		if !bytes.Equal(info.GetPubKey().Address(), submitter) {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, "Must use own submitter address")
			return
		}

		if req.Headers == nil || len(req.Headers) != 2 {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, "Must have 2 headers exactly")
			return
		}

		msg := slashing.NewMsgBscSubmitEvidence(submitter, req.Headers)

		txBldr := authtxb.TxBuilder{
			Codec:   cdc,
			ChainID: baseReq.ChainID,
		}
		txBldr = txBldr.WithAccountNumber(baseReq.AccountNumber).WithSequence(baseReq.Sequence)
		baseReq.Sequence++

		if utils.HasDryRunArg(r) {
			// Todo return something
			return
		}

		if utils.HasGenerateOnlyArg(r) {
			utils.WriteGenerateStdTxResponse(w, txBldr, []sdk.Msg{msg})
			return
		}

		txBytes, err := txBldr.BuildAndSign(baseReq.Name, baseReq.Password, []sdk.Msg{msg})
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}

		res, err := cliCtx.BroadcastTx(txBytes)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		utils.PostProcessResponse(w, cdc, res, cliCtx.Indent)
	}

}
