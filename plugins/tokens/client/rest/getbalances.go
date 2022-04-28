package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/plugins/tokens"
	"github.com/bnb-chain/node/wire"
)

// BalanceReqHandler creates an http request handler to get the token balances of a given address
func BalancesReqHandler(
	cdc *wire.Codec, ctx context.CLIContext, tokens tokens.Mapper,
) http.HandlerFunc {
	type response struct {
		Address  string         `json:"address"`
		Balances []TokenBalance `json:"balances"`
	}
	responseType := "application/json"

	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(err.Error()))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		// collect params
		// convert bech32 address
		addr, err := sdk.AccAddressFromBech32(vars["address"])
		if err != nil {
			throw(w, http.StatusBadRequest, err)
			return
		}

		bals, err := GetBalances(cdc, ctx, tokens, addr)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		resp := response{
			Address:  vars["address"],
			Balances: bals,
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(output)
	}
}
