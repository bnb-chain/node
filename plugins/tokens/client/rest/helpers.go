package rest

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

type TokenBalance struct {
	Symbol string       `json:"symbol"`
	Free   utils.Fixed8 `json:"free"`
	Locked utils.Fixed8 `json:"locked"`
	Frozen utils.Fixed8 `json:"frozen"`
}

func GetBalances(
	cdc *wire.Codec, ctx context.CLIContext, tokens tokens.Mapper, addr sdk.AccAddress,
) ([]TokenBalance, error) {
	coins, err := getCoinsCC(cdc, ctx, addr)
	if err != nil {
		return nil, err
	}

	// must do it this way because GetTokenList relies on store.Iterator
	// which we can't use from a CLIContext
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
	bals := make([]TokenBalance, 0, len(denoms))
	for symb := range denoms {
		symbs = append(symbs, symb)
		// count locked and frozen coins
		var locked, frozen int64
		lockedc, err := getLockedCC(cdc, ctx, addr)
		if err != nil {
			fmt.Println("getLockedCC error ignored, will use `0`")
		} else {
			locked = lockedc.AmountOf(symb)
		}
		frozenc, err := getFrozenCC(cdc, ctx, addr)
		if err != nil {
			fmt.Println("getFrozenCC error ignored, will use `0`")
		} else {
			frozen = frozenc.AmountOf(symb)
		}
		bals = append(bals, TokenBalance{
			Symbol: symb,
			Free:   utils.Fixed8(coins.AmountOf(symb)),
			Locked: utils.Fixed8(locked),
			Frozen: utils.Fixed8(frozen),
		})
	}

	return bals, nil
}

func decodeAccount(cdc *wire.Codec, bz *[]byte) (acc auth.Account, err error) {
	err = cdc.UnmarshalBinaryBare(*bz, &acc)
	if err != nil {
		return nil, err
	}
	return acc, err
}

func getAccount(cdc *wire.Codec, ctx context.CLIContext, addr sdk.AccAddress) (auth.Account, error) {
	key := auth.AddressStoreKey(addr)
	bz, err := ctx.QueryStore(key, common.AccountStoreName)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, nil
	}
	acc, err := decodeAccount(cdc, &bz)
	return acc, err
}

func getCoinsCC(cdc *wire.Codec, ctx context.CLIContext, addr sdk.AccAddress) (sdk.Coins, error) {
	acc, err := getAccount(cdc, ctx, addr)
	if err != nil {
		return sdk.Coins{}, err
	}
	if acc == nil {
		return sdk.Coins{}, nil
	}
	return acc.GetCoins(), nil
}

func getLockedCC(cdc *wire.Codec, ctx context.CLIContext, addr sdk.AccAddress) (sdk.Coins, error) {
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

func getFrozenCC(cdc *wire.Codec, ctx context.CLIContext, addr sdk.AccAddress) (sdk.Coins, error) {
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
