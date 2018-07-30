package rest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

// RegisterBalanceRoute registers this http route handler
func RegisterBalanceRoute(
	ctx context.CoreContext,
	r *mux.Router,
	cdc *wire.Codec,
	tokens tokens.Mapper,
) *mux.Route {
	return r.HandleFunc("/balances/{address}/{symbol}", balanceRequestHandler(cdc, tokens, ctx)).Methods("GET")
}

type balanceResponse struct {
	Address string       `json:"address"`
	Balance TokenBalance `json:"balance"`
}

func balanceRequestHandler(cdc *wire.Codec, tokens tokens.Mapper, ctx context.CoreContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		throw := func(status int, err error) {
			w.WriteHeader(status)
			w.Write([]byte(err.Error()))
			return
		}

		vars := mux.Vars(r)

		// collect params
		// convert bech32 address
		addr, err := sdk.AccAddressFromBech32(vars["address"])
		if err != nil {
			throw(http.StatusBadRequest, err)
			return
		}
		params := struct {
			address sdk.AccAddress
			symbol  string
		}{
			address: addr,
			symbol:  vars["symbol"],
		}

		// exists := tokens.ExistsCC(ctx, params.symbol)
		exists := true
		if !exists {
			throw(http.StatusNotFound, errors.New("symbol not found"))
			return
		}

		// coins := bank.GetCoins(ctx, params.address)
		coins, err := getCoinsCC(cdc, ctx, params.address)
		if err != nil {
			throw(http.StatusNotFound, err)
			return
		}

		locked := sdk.NewInt(0)
		frozen := sdk.NewInt(0)
		lockedc, err := getLockedCC(cdc, ctx, params.address)
		if err != nil {
			locked = lockedc.AmountOf(params.symbol)
		}
		frozenc, err := getFrozenCC(cdc, ctx, params.address)
		if err != nil {
			frozen = frozenc.AmountOf(params.symbol)
		}

		fmt.Println(utils.Fixed8(coins.AmountOf(params.symbol).Int64()))

		resp := balanceResponse{
			Address: vars["address"],
			Balance: TokenBalance{
				Symbol: params.symbol,
				Free:   utils.Fixed8(coins.AmountOf(params.symbol).Int64()),
				Locked: utils.Fixed8(locked.Int64()),
				Frozen: utils.Fixed8(frozen.Int64()),
			},
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(http.StatusInternalServerError, err)
			return
		}

		w.Write(output)
	}
}
