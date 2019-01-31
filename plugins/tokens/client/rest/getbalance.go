package rest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"
)

// BalanceReqHandler creates an http request handler to get an individual token balance of a given address
func BalanceReqHandler(cdc *wire.Codec, ctx context.CLIContext, tokens tokens.Mapper) http.HandlerFunc {
	type params struct {
		address sdk.AccAddress
		symbol  string
	}
	type response struct {
		Address string       `json:"address"`
		Balance TokenBalance `json:"balance"`
	}
	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
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
			symbol:  vars["symbol"],
		}

		exists := tokens.ExistsCC(ctx, params.symbol)
		if !exists {
			throw(w, http.StatusNotFound, errors.New("symbol not found"))
			return
		}

		coins, err := getCoinsCC(cdc, ctx, params.address)
		if err != nil {
			throw(w, http.StatusNotFound, err)
			return
		}

		// count locked and frozen coins
		var locked, frozen int64
		lockedc, err := getLockedCC(cdc, ctx, params.address)
		if err != nil {
			fmt.Println("getLockedCC error ignored, will use `0`")
		} else {
			locked = lockedc.AmountOf(params.symbol)
		}
		frozenc, err := getFrozenCC(cdc, ctx, params.address)
		if err != nil {
			fmt.Println("getFrozenCC error ignored, will use `0`")
		} else {
			frozen = frozenc.AmountOf(params.symbol)
		}

		resp := response{
			Address: vars["address"],
			Balance: TokenBalance{
				Symbol: params.symbol,
				Free:   utils.Fixed8(coins.AmountOf(params.symbol)),
				Locked: utils.Fixed8(locked),
				Frozen: utils.Fixed8(frozen),
			},
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		w.Write(output)
	}
}
