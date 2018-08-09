package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

// BalanceReqHandler creates an http request handler to get the token balances of a given address
func BalancesReqHandler(
	cdc *wire.Codec, ctx context.CoreContext, tokens tokens.Mapper,
) http.HandlerFunc {
	type params struct {
		address sdk.AccAddress
	}
	type response struct {
		Address  string         `json:"address"`
		Balances []tokenBalance `json:"balances"`
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
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

		params := params{
			address: addr,
		}

		coins, err := getCoinsCC(cdc, ctx, params.address)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		// must do it this way because GetTokenList relies on store.Iterator
		// which we can't use from a CoreContext
		var denoms map[string]bool
		denoms = map[string]bool{}
		for _, coin := range coins {
			denom := coin.Denom
			exists := tokens.ExistsCC(ctx, denom)
			// TODO: we probably actually want to show zero balances.
			// if exists && !sdk.Int.IsZero(coins.AmountOf(denom)) {
			if exists {
				denoms[denom] = true
			}
		}

		symbs := make([]string, 0, len(denoms))
		bals := make([]tokenBalance, 0, len(denoms))
		for symb := range denoms {
			symbs = append(symbs, symb)
			// count locked and frozen coins
			locked := sdk.NewInt(0)
			frozen := sdk.NewInt(0)
			lockedc, err := getLockedCC(cdc, ctx, params.address)
			if err != nil {
				fmt.Println("getLockedCC error ignored, will use `0`")
			} else {
				locked = lockedc.AmountOf(symb)
			}
			frozenc, err := getFrozenCC(cdc, ctx, params.address)
			if err != nil {
				fmt.Println("getFrozenCC error ignored, will use `0`")
			} else {
				frozen = frozenc.AmountOf(symb)
			}
			bals = append(bals, tokenBalance{
				Symbol: symb,
				Free:   utils.Fixed8(coins.AmountOf(symb).Int64()),
				Locked: utils.Fixed8(locked.Int64()),
				Frozen: utils.Fixed8(frozen.Int64()),
			})
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

		w.Write(output)
	}
}
