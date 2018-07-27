package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

type balancesSendBody struct {
	Address Address `json:"address"`
}

type TokenBalance struct {
	Symbol  string  `json:"symbol"`
	Balance sdk.Int `json:"free"`
	Locked  sdk.Int `json:"locked"`
	Frozen  sdk.Int `json:"frozen"`
}

type balancesResponse struct {
	Address  string         `json:"address"`
	Balances []TokenBalance `json:"balances"`
}

// RegisterRoutes - Central function to define routes that get registered by the main application
func registerTokensRoute(
	ctx context.CoreContext,
	r *mux.Router,
	cdc *wire.Codec,
	tokens tokens.Mapper,
) {
	r.HandleFunc("/balances/{address}", TokensRequestHandler(cdc, tokens, ctx)).Methods("GET")
}

// temporary account decoder bits

func decodeAccount(cdc *wire.Codec, bz []byte) (acc auth.Account, err error) {
	err = cdc.UnmarshalBinaryBare(bz, &acc)
	if err != nil {
		return nil, err
	}
	return acc, err
}

func getAccount(cdc *wire.Codec, ctx context.CoreContext, addr sdk.AccAddress) (auth.Account, error) {
	key := auth.AddressStoreKey(addr)
	bz, err := ctx.QueryStore(key, common.AccountStoreName)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, nil
	}
	acc, err := decodeAccount(cdc, bz)
	return acc, err
}

func getCoinsCC(cdc *wire.Codec, ctx context.CoreContext, addr sdk.AccAddress) (sdk.Coins, error) {
	acc, err := getAccount(cdc, ctx, addr)
	if err != nil {
		return sdk.Coins{}, err
	}
	if acc == nil {
		return sdk.Coins{}, nil
	}
	return acc.GetCoins(), nil
}

func getLockedCC(cdc *wire.Codec, ctx context.CoreContext, addr sdk.AccAddress) (sdk.Coins, error) {
	acc, err := getAccount(cdc, ctx, addr)
	nacc := acc.(types.NamedAccount)
	if err != nil {
		return sdk.Coins{}, err
	}
	if acc == nil {
		return sdk.Coins{}, nil
	}
	return nacc.GetLockedCoins(), nil
}

func getFrozenCC(cdc *wire.Codec, ctx context.CoreContext, addr sdk.AccAddress) (sdk.Coins, error) {
	acc, err := getAccount(cdc, ctx, addr)
	nacc := acc.(types.NamedAccount)
	if err != nil {
		return sdk.Coins{}, err
	}
	if acc == nil {
		return sdk.Coins{}, nil
	}
	return nacc.GetFrozenCoins(), nil
}

// end temp stuff

// TokensRequestHandler - http request handler to send coins to a address
func TokensRequestHandler(
	cdc *wire.Codec, tokens tokens.Mapper, ctx context.CoreContext,
) http.HandlerFunc {
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
		}{
			address: addr,
		}

		// coins := bank.GetCoins(ctx, params.address)

		coins, err := getCoinsCC(cdc, ctx, params.address)
		if err != nil {
			throw(http.StatusNotFound, err)
			return
		}

		// must do it this way because GetTokenList relies on store.Iterator
		// which we can't use from a CoreContext
		var denoms map[string]bool
		denoms = map[string]bool{}
		for _, coin := range coins {
			denom := coin.Denom
			exists := true
			// exists := tokens.ExistsCC(ctx, denom)
			// TODO: we probably actually want to show zero balances.
			// if exists && !sdk.Int.IsZero(coins.AmountOf(denom)) {
			if exists {
				denoms[denom] = true
			}
		}

		symbs := make([]string, 0, len(denoms))
		bals := make([]TokenBalance, 0, len(denoms))
		for symb := range denoms {
			symbs = append(symbs, symb)
			locked := sdk.NewInt(0)
			frozen := sdk.NewInt(0)
			lockedc, err := getLockedCC(cdc, ctx, params.address)
			if err != nil {
				locked = lockedc.AmountOf(symb)
			}
			frozenc, err := getFrozenCC(cdc, ctx, params.address)
			if err != nil {
				frozen = frozenc.AmountOf(symb)
			}
			bals = append(bals, TokenBalance{
				Symbol:  symb,
				Balance: coins.AmountOf(symb),
				Locked:  locked,
				Frozen:  frozen,
			})
		}

		resp := balancesResponse{
			Address:  vars["address"],
			Balances: bals,
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(http.StatusInternalServerError, err)
			return
		}

		w.Write(output)
	}
}
