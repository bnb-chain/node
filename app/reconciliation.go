package app

import (
	"encoding/binary"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/common"
	"github.com/bnb-chain/node/common/types"
)

const globalAccountNumber = "globalAccountNumber"

// unbalancedBlockHeightKey for saving unbalanced block height for reconciliation
var unbalancedBlockHeightKey = []byte("0x01")

// reconBalance will do reconciliation for accounts balances.
func (app *BinanceChain) reconBalance(ctx sdk.Context, accountIavl *store.IavlStore, tokenIavl *store.IavlStore) {
	height, exists := app.getUnbalancedBlockHeight(ctx)
	if exists {
		panic(fmt.Sprintf("unbalanced state at block height %d, please use hardfork to bypass it", height))
	}

	accPre, accCurrent := app.getAccountChanges(ctx, accountIavl)
	tokenPre, tokenCurrent := app.getTokenChanges(ctx, tokenIavl)

	// accPre and tokenPre are positive, there will be no overflow
	accountDiff := accCurrent.Plus(accPre.Negative())
	tokenDiff := tokenCurrent.Plus(tokenPre.Negative())

	if !accountDiff.IsEqual(tokenDiff) {
		ctx.Logger().Error(fmt.Sprintf("unbalanced at block %d, account diff: %s, token diff: %s \n",
			ctx.BlockHeight(), accountDiff.String(), tokenDiff.String()))
		app.saveUnbalancedBlockHeight(ctx)
	}
}

func (app *BinanceChain) getAccountChanges(ctx sdk.Context, accountStore *store.IavlStore) (sdk.Coins, sdk.Coins) {
	preCoins := sdk.Coins{}
	currentCoins := sdk.Coins{}

	diff := accountStore.GetDiff()
	version := accountStore.GetTree().Version() - 1
	for k := range diff {
		if k == globalAccountNumber {
			continue
		}
		v := accountStore.Get([]byte(k))
		if v != nil {
			var acc1 sdk.Account
			err := app.Codec.UnmarshalBinaryBare(v, &acc1)
			if err != nil {
				panic("failed to unmarshal current value " + err.Error())
			}
			nacc1 := acc1.(types.NamedAccount)
			ctx.Logger().Debug("current account", "address", nacc1.GetAddress(), "coins", nacc1.GetCoins().String())
			currentCoins = currentCoins.Plus(nacc1.GetCoins())
			currentCoins = currentCoins.Plus(nacc1.GetFrozenCoins())
			currentCoins = currentCoins.Plus(nacc1.GetLockedCoins())
		}

		_, v = accountStore.GetTree().GetVersioned([]byte(k), version)
		if v != nil { // it is not a new account
			var acc2 sdk.Account
			err := app.Codec.UnmarshalBinaryBare(v, &acc2)
			if err != nil {
				panic("failed to unmarshal previous value " + err.Error())
			}
			nacc2 := acc2.(types.NamedAccount)
			ctx.Logger().Debug("previous account", "address", nacc2.GetAddress(), "coins", nacc2.GetCoins().String())
			preCoins = preCoins.Plus(nacc2.GetCoins())
			preCoins = preCoins.Plus(nacc2.GetFrozenCoins())
			preCoins = preCoins.Plus(nacc2.GetLockedCoins())
		}
	}
	ctx.Logger().Debug("account changes", "current", currentCoins.String(), "previous", preCoins.String(),
		"version", version, "height", ctx.BlockHeight())

	return preCoins, currentCoins
}

func (app *BinanceChain) getTokenChanges(ctx sdk.Context, tokenStore *store.IavlStore) (sdk.Coins, sdk.Coins) {
	preCoins := sdk.Coins{}
	currentCoins := sdk.Coins{}

	diff := tokenStore.GetDiff()
	version := tokenStore.GetTree().Version() - 1
	for k := range diff {
		v := tokenStore.Get([]byte(k))
		if v != nil {
			var token1 types.IToken
			err := app.Codec.UnmarshalBinaryBare(v, &token1)
			if err != nil {
				panic("failed to unmarshal current value " + err.Error())
			}
			ctx.Logger().Debug("current token", "symbol", token1.GetSymbol(), "supply", token1.GetTotalSupply().ToInt64())
			currentCoins = currentCoins.Plus(sdk.Coins{
				sdk.NewCoin(token1.GetSymbol(), token1.GetTotalSupply().ToInt64()),
			})
		}

		_, v = tokenStore.GetTree().GetVersioned([]byte(k), version)
		if v != nil { // it is not a new token
			var token2 types.IToken
			err := app.Codec.UnmarshalBinaryBare(v, &token2)
			if err != nil {
				panic("failed to unmarshal previous value " + err.Error())
			}
			ctx.Logger().Debug("previous token", "symbol", token2.GetSymbol(), "supply", token2.GetTotalSupply().ToInt64())
			preCoins = preCoins.Plus(sdk.Coins{
				sdk.NewCoin(token2.GetSymbol(), token2.GetTotalSupply().ToInt64()),
			})
		}
	}
	ctx.Logger().Debug("token changes", "current", currentCoins.String(), "previous", preCoins.String(),
		"version", version, "height", ctx.BlockHeight())

	return preCoins, currentCoins
}

func (app *BinanceChain) saveUnbalancedBlockHeight(ctx sdk.Context) {
	reconStore := app.GetCommitMultiStore().GetCommitStore(common.ReconStoreKey).(*store.IavlStore)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz[:], uint64(ctx.BlockHeight()))
	reconStore.Set(unbalancedBlockHeightKey, bz)
}

func (app *BinanceChain) getUnbalancedBlockHeight(ctx sdk.Context) (uint64, bool) {
	reconStore := app.GetCommitMultiStore().GetCommitStore(common.ReconStoreKey).(*store.IavlStore)

	bz := reconStore.Get(unbalancedBlockHeightKey)
	if bz == nil {
		return 0, false
	}
	return binary.BigEndian.Uint64(bz), true
}
