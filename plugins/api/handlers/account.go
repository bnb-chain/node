package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"

	"github.com/BiJie/BinanceChain/common/types"
	tkclient "github.com/BiJie/BinanceChain/plugins/tokens/client/rest"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/BiJie/BinanceChain/wire"
)

// AccountReqHandler queries for an account and returns its information.
func AccountReqHandler(
	cdc *wire.Codec, ctx context.CoreContext, tokens tkstore.Mapper, accStoreName string,
) http.HandlerFunc {
	type response struct {
		auth.BaseAccount
		Balances []tkclient.TokenBalance `json:"balances"`
		Coins    *struct{}               `json:"coins,omitempty"` // omit `coins`
	}

	responseType := "application/json"

	accDecoder := authcmd.GetAccountDecoder(cdc)

	throw := func(w http.ResponseWriter, status int, message string) {
		w.WriteHeader(status)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(message))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bech32addr := vars["address"]

		addr, err := sdk.AccAddressFromBech32(bech32addr)
		if err != nil {
			throw(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := ctx.QueryStore(auth.AddressStoreKey(addr), accStoreName)
		if err != nil {
			errMsg := fmt.Sprintf("couldn't query account. Error: %s", err.Error())
			throw(w, http.StatusInternalServerError, errMsg)
			return
		}

		// the query will return empty if there is no data for this account
		if len(res) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// decode the value
		account, err := accDecoder(res)
		if err != nil {
			errMsg := fmt.Sprintf("couldn't parse query result. Result: %s. Error: %s", res, err.Error())
			throw(w, http.StatusInternalServerError, errMsg)
			return
		}

		bals, err := tkclient.GetBalances(cdc, ctx, tokens, account.GetAddress())
		resp := response{
			BaseAccount: account.(*types.AppAccount).BaseAccount,
			Balances:    bals,
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", responseType)
		json.NewEncoder(w).Encode(resp)
	}
}
