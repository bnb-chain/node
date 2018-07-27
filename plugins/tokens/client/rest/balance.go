package rest

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"

	"github.com/BiJie/BinanceChain/plugins/tokens"
)

// https://github.com/tendermint/tendermint/blob/05a76fb517f50da27b4bfcdc7b4cf185fc61eff6/crypto/crypto.go#L14
// RegisterRoutes - Central function to define routes that get registered by the main application
func registerBalanceRoute(ctx context.CoreContext, r *mux.Router, cdc *wire.Codec, tokens tokens.Mapper) {
	r.HandleFunc("/balances/{address}/{symbol}", BalanceRequestHandler(cdc, tokens, ctx)).Methods("GET")
}

type balanceSendBody struct {
	Address Address `json:"address"`
	Symbol  string  `json:"symbol"`
}

type balanceResponse struct {
	Address string       `json:"address"`
	Balance TokenBalance `json:"balance"`
}

// BalanceRequestHandler - http request handler to send coins to a address
func BalanceRequestHandler(cdc *wire.Codec, tokens tokens.Mapper, ctx context.CoreContext) http.HandlerFunc {
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

		resp := balanceResponse{
			Address: vars["address"],
			Balance: TokenBalance{
				Symbol:  params.symbol,
				Balance: coins.AmountOf(params.symbol),
				Locked:  locked,
				Frozen:  frozen,
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
