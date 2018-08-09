package rest

import (
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/wire"
)

type tokenBalance struct {
	Symbol string       `json:"symbol"`
	Free   utils.Fixed8 `json:"free"`
	Locked utils.Fixed8 `json:"locked"`
	Frozen utils.Fixed8 `json:"frozen"`
}

func decodeAccount(cdc *wire.Codec, bz *[]byte) (acc auth.Account, err error) {
	err = cdc.UnmarshalBinaryBare(*bz, &acc)
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
	acc, err := decodeAccount(cdc, &bz)
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
