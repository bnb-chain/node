package app

import (
	"fmt"
	"github.com/bnb-chain/node/common"
	"github.com/bnb-chain/node/common/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const globalAccountNumber = "globalAccountNumber"

// reconBalance will do reconciliation for accounts balances.
func (app *BinanceChain) reconBalance(ctx sdk.Context) {
	currentHeight := ctx.BlockHeight()

	accPre, accCurrent := app.getAccountChanges(ctx)
	tokenPre, tokenCurrent := app.getTokenChanges(ctx)

	left := tokenPre.Plus(accCurrent)
	right := tokenCurrent.Plus(accPre)

	if !left.IsEqual(right) {
		err := fmt.Sprintf("unbalanced at block %d, pre: %s, current: %s \n",
			currentHeight, left.String(), right.String())
		ctx.Logger().Error(err)
		panic(err)
	}
}

func (app *BinanceChain) getAccountChanges(ctx sdk.Context) (sdk.Coins, sdk.Coins) {
	iavlStore, ok := app.GetCommitMultiStore().GetCommitStore(common.AccountStoreKey).(*store.IavlStore)
	if !ok {
		panic("cannot convert to ival store")
	}

	preCoins := sdk.Coins{}
	currentCoins := sdk.Coins{}

	diff := iavlStore.GetDiff()
	version := iavlStore.GetTree().Version() - 1
	for k, v := range diff {
		if k == globalAccountNumber {
			continue
		}
		var acc1 sdk.Account
		err := app.Codec.UnmarshalBinaryBare(v, &acc1)
		if err != nil {
			panic("failed to unmarshal diff value " + err.Error())
		}
		nacc1 := acc1.(types.NamedAccount)
		ctx.Logger().Debug("diff account", "address", nacc1.GetAddress(), "coins", nacc1.GetCoins().String())
		currentCoins = currentCoins.Plus(nacc1.GetCoins())
		currentCoins = currentCoins.Plus(nacc1.GetFrozenCoins())
		currentCoins = currentCoins.Plus(nacc1.GetLockedCoins())

		var acc2 sdk.Account
		_, v = iavlStore.GetTree().GetVersioned([]byte(k), version)
		if v != nil { // it is not a new account
			err = app.Codec.UnmarshalBinaryBare(v, &acc2)
			if err != nil {
				panic("failed to unmarshal previous value " + err.Error())
			}
			nacc2 := acc2.(types.NamedAccount)

			ctx.Logger().Debug("pre account", "address", nacc2.GetAddress(), "coins", nacc2.GetCoins().String())
			preCoins = preCoins.Plus(nacc2.GetCoins())
			preCoins = preCoins.Plus(nacc2.GetFrozenCoins())
			preCoins = preCoins.Plus(nacc2.GetLockedCoins())
		}
	}
	iavlStore.ResetDiff()

	ctx.Logger().Debug("account changes", "diff", currentCoins.String(), "previous", preCoins.String(), "height", ctx.BlockHeight())

	return preCoins, currentCoins
}

func (app *BinanceChain) getTokenChanges(ctx sdk.Context) (sdk.Coins, sdk.Coins) {
	iavlStore, ok := app.GetCommitMultiStore().GetCommitStore(common.TokenStoreKey).(*store.IavlStore)
	if !ok {
		panic("cannot convert to ival store")
	}

	preCoins := sdk.Coins{}
	currentCoins := sdk.Coins{}

	diff := iavlStore.GetDiff()
	version := iavlStore.GetTree().Version() - 1
	for k, v := range diff {
		var token1 types.IToken
		err := app.Codec.UnmarshalBinaryBare(v, &token1)
		if err != nil {
			panic("failed to unmarshal diff value " + err.Error())
		}
		currentCoins = currentCoins.Plus(sdk.Coins{
			sdk.NewCoin(token1.GetSymbol(), token1.GetTotalSupply().ToInt64()),
		})

		var token2 types.IToken
		_, v = iavlStore.GetTree().GetVersioned([]byte(k), version)
		if v != nil { // it is not a new token
			err = app.Codec.UnmarshalBinaryBare(v, &token2)
			if err != nil {
				panic("failed to unmarshal previous value " + err.Error())
			}
			preCoins = preCoins.Plus(sdk.Coins{
				sdk.NewCoin(token2.GetSymbol(), token2.GetTotalSupply().ToInt64()),
			})
		}
	}
	iavlStore.ResetDiff()

	ctx.Logger().Debug("token changes", "diff", currentCoins.String(), "previous", preCoins.String(), "height", ctx.BlockHeight())

	return preCoins, currentCoins
}
