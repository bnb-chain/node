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

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	tkclient "github.com/binance-chain/node/plugins/tokens/client/rest"
	"github.com/binance-chain/node/wire"
)

// AccountReqHandler queries for an account and returns its information.
func AccountReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	type response struct {
		auth.BaseAccount
		Balances []tkclient.TokenBalance `json:"balances"`
		Coins    *struct{}               `json:"coins,omitempty"` // omit `coins`
	}

	responseType := "application/json"

	accDecoder := authcmd.GetAccountDecoder(cdc)

	throw := func(w http.ResponseWriter, status int, message string) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(message))
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bech32addr := vars["address"]

		_, err := sdk.AccAddressFromBech32(bech32addr)
		if err != nil {
			throw(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := ctx.Query(fmt.Sprintf("/account/%s", bech32addr), nil)
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

		appAccount := account.(*types.AppAccount)
		resp := response{
			BaseAccount: appAccount.BaseAccount,
			Balances:    toTokenBalances(appAccount),
		}

		w.Header().Set("Content-Type", responseType)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func toTokenBalances(acc *types.AppAccount) []tkclient.TokenBalance {
	balances := make(map[string]tkclient.TokenBalance)
	for _, coin := range acc.GetCoins() {
		balances[coin.Denom] = tkclient.TokenBalance{Symbol: coin.Denom, Free: utils.Fixed8(coin.Amount)}
	}

	for _, coin := range acc.GetLockedCoins() {
		if balance, ok := balances[coin.Denom]; ok {
			balance.Locked = utils.Fixed8(coin.Amount)
		} else {
			balances[coin.Denom] = tkclient.TokenBalance{Symbol: coin.Denom, Locked: utils.Fixed8(coin.Amount)}
		}
	}

	for _, coin := range acc.GetFrozenCoins() {
		if balance, ok := balances[coin.Denom]; ok {
			balance.Frozen = utils.Fixed8(coin.Amount)
		} else {
			balances[coin.Denom] = tkclient.TokenBalance{Symbol: coin.Denom, Frozen: utils.Fixed8(coin.Amount)}
		}
	}

	res := make([]tkclient.TokenBalance, len(balances), len(balances))
	i := 0
	for _, balance := range balances {
		res[i] = balance
		i++
	}
	return res
}
