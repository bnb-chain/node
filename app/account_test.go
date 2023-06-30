package app

import (
	"io"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	common "github.com/bnb-chain/node/common/types"
)

func BenchmarkGetAccount(b *testing.B) {
	memDB := db.NewMemDB()
	logger := log.NewTMLogger(io.Discard)
	testApp := NewBNBBeaconChain(logger, memDB, io.Discard)

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

	acc.SetCoins(sdk.Coins{sdk.NewCoin("BNB", 1000), sdk.NewCoin("BTC", 1000), sdk.NewCoin("ETH", 100)})
	acc.SetLockedCoins(sdk.Coins{sdk.NewCoin("BNB", 1000), sdk.NewCoin("BTC", 1000), sdk.NewCoin("ETH", 100)})
	acc.SetFrozenCoins(sdk.Coins{sdk.NewCoin("BNB", 1000), sdk.NewCoin("BTC", 1000), sdk.NewCoin("ETH", 100)})

	testApp.AccountKeeper.SetAccount(ctx, acc)
	for i := 0; i < b.N; i++ {
		_ = testApp.AccountKeeper.GetAccount(ctx, baseAcc.Address).(common.NamedAccount)
	}
}

func BenchmarkSetAccount(b *testing.B) {
	memDB := db.NewMemDB()
	logger := log.NewTMLogger(io.Discard)
	testApp := NewBNBBeaconChain(logger, memDB, io.Discard)

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

	acc.SetCoins(sdk.Coins{sdk.NewCoin("BNB", 1000), sdk.NewCoin("BTC", 1000), sdk.NewCoin("ETH", 100)})
	acc.SetLockedCoins(sdk.Coins{sdk.NewCoin("BNB", 1000), sdk.NewCoin("BTC", 1000), sdk.NewCoin("ETH", 100)})
	acc.SetFrozenCoins(sdk.Coins{sdk.NewCoin("BNB", 1000), sdk.NewCoin("BTC", 1000), sdk.NewCoin("ETH", 100)})

	for i := 0; i < b.N; i++ {
		testApp.AccountKeeper.SetAccount(ctx, acc)
	}
}
