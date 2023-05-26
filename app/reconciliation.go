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
func (app *BinanceChain) reconBalance(ctx sdk.Context) error {
	currentHeight := ctx.BlockHeight()

	ctx.Logger().Debug("account changes")
	accPre, accCurrent := app.getAccountChanges(ctx)

	ctx.Logger().Debug("token changes")
	tokenPre, tokenCurrent := app.getTokenChanges(ctx)

	left := tokenPre.Plus(accCurrent)
	right := tokenCurrent.Plus(accPre)

	if !left.IsEqual(right) {
		err := fmt.Sprintf("unbalanced at block %d, pre: %s, current: %s \n",
			currentHeight, left.String(), right.String())
		ctx.Logger().Error(err)
		panic(err)
	}
	return nil
}

func (app *BinanceChain) getAccountChanges(ctx sdk.Context) (sdk.Coins, sdk.Coins) {
	currentHeight := ctx.BlockHeight()
	iavlStore, ok := app.GetCommitMultiStore().GetCommitStore(common.AccountStoreKey).(*store.IavlStore)
	if !ok {
		panic("cannot convert to ival")
	}

	preVersion := currentHeight - 2
	preCoins := sdk.Coins{}
	currentCoins := sdk.Coins{}

	diff := iavlStore.GetDiff()
	for k, v := range diff {
		if k == globalAccountNumber {
			continue
		}
		var acc1 sdk.Account
		err := app.Codec.UnmarshalBinaryBare(v, &acc1)
		if err != nil {
			panic("failed to unmarshal diff value " + err.Error())
		}
		currentCoins = currentCoins.Plus(acc1.GetCoins())

		var acc2 sdk.Account
		_, v = iavlStore.GetTree().GetVersioned([]byte(k), preVersion)
		if v != nil { // it is not a new account
			err = app.Codec.UnmarshalBinaryBare(v, &acc2)
			if err != nil {
				panic("failed to unmarshal previous value " + err.Error())
			}
			preCoins = preCoins.Plus(acc2.GetCoins())
		}
	}

	if len(currentCoins) > 0 {
		ctx.Logger().Debug("diff coins", "coins", currentCoins.String())
	}
	if len(preCoins) > 0 {
		ctx.Logger().Debug("previous coins", "coins", preCoins.String())
	}

	return preCoins, currentCoins
}

func (app *BinanceChain) getTokenChanges(ctx sdk.Context) (sdk.Coins, sdk.Coins) {
	currentHeight := ctx.BlockHeight()

	iavlStore, ok := app.GetCommitMultiStore().GetCommitStore(common.TokenStoreKey).(*store.IavlStore)
	if !ok {
		panic("cannot convert to ival")
	}

	preVersion := currentHeight - 2
	preCoins := sdk.Coins{}
	currentCoins := sdk.Coins{}

	diff := iavlStore.GetDiff()
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
		_, v = iavlStore.GetTree().GetVersioned([]byte(k), preVersion)
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

	if len(currentCoins) > 0 {
		ctx.Logger().Debug("diff coins", "coins", currentCoins.String())
	}
	if len(preCoins) > 0 {
		ctx.Logger().Debug("previous coins", "coins", preCoins.String())
	}

	return preCoins, currentCoins
}
