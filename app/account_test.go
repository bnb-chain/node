package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/golang/go/src/io/ioutil"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	common "github.com/BiJie/BinanceChain/common/types"
)

func BenchmarkGetAccount(b *testing.B) {
	memDB := db.NewMemDB()
	logger := log.NewTMLogger(ioutil.Discard)
	testApp := NewBinanceChain(logger, memDB, ioutil.Discard)

	pk := ed25519.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pk.Address())
	baseAcc := auth.BaseAccount{
		Address: addr,
	}

	ctx := testApp.BaseApp.NewContext(sdk.RunTxModeCheck, abci.Header{})

	acc := &common.AppAccount{
		BaseAccount: auth.BaseAccount{
			Address: baseAcc.GetAddress(),
			Coins:   baseAcc.GetCoins(),
		}}
	if testApp.AccountKeeper.GetAccount(ctx, acc.GetAddress()) == nil {
		acc.BaseAccount.AccountNumber = testApp.AccountKeeper.GetNextAccountNumber(ctx)
	}

	acc.SetCoins(sdk.Coins{sdk.NewInt64Coin("BNB", 1000), sdk.NewInt64Coin("BTC", 1000), sdk.NewInt64Coin("ETH", 100)})
	acc.SetLockedCoins(sdk.Coins{sdk.NewInt64Coin("BNB", 1000), sdk.NewInt64Coin("BTC", 1000), sdk.NewInt64Coin("ETH", 100)})
	acc.SetFrozenCoins(sdk.Coins{sdk.NewInt64Coin("BNB", 1000), sdk.NewInt64Coin("BTC", 1000), sdk.NewInt64Coin("ETH", 100)})

	testApp.AccountKeeper.SetAccount(ctx, acc)
	for i := 0; i < b.N; i++ {
		_ = testApp.AccountKeeper.GetAccount(ctx, baseAcc.Address).(common.NamedAccount)
	}
}

func BenchmarkSetAccount(b *testing.B) {
	memDB := db.NewMemDB()
	logger := log.NewTMLogger(ioutil.Discard)
	testApp := NewBinanceChain(logger, memDB, ioutil.Discard)

	pk := ed25519.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pk.Address())
	baseAcc := auth.BaseAccount{
		Address: addr,
	}

	ctx := testApp.BaseApp.NewContext(sdk.RunTxModeCheck, abci.Header{})

	acc := &common.AppAccount{
		BaseAccount: auth.BaseAccount{
			Address: baseAcc.GetAddress(),
			Coins:   baseAcc.GetCoins(),
		}}
	if testApp.AccountKeeper.GetAccount(ctx, acc.GetAddress()) == nil {
		acc.BaseAccount.AccountNumber = testApp.AccountKeeper.GetNextAccountNumber(ctx)
	}

	acc.SetCoins(sdk.Coins{sdk.NewInt64Coin("BNB", 1000), sdk.NewInt64Coin("BTC", 1000), sdk.NewInt64Coin("ETH", 100)})
	acc.SetLockedCoins(sdk.Coins{sdk.NewInt64Coin("BNB", 1000), sdk.NewInt64Coin("BTC", 1000), sdk.NewInt64Coin("ETH", 100)})
	acc.SetFrozenCoins(sdk.Coins{sdk.NewInt64Coin("BNB", 1000), sdk.NewInt64Coin("BTC", 1000), sdk.NewInt64Coin("ETH", 100)})

	for i := 0; i < b.N; i++ {
		testApp.AccountKeeper.SetAccount(ctx, acc)
	}
}
