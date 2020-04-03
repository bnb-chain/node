package testutils

import (
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	dbm "github.com/tendermint/tendermint/libs/db"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/types"
)

func SetupMultiStoreForUnitTest() (sdk.MultiStore, *sdk.KVStoreKey, *sdk.KVStoreKey) {
	_, ms, capKey, capKey2, _ := SetupMultiStoreWithDBForUnitTest()
	return ms, capKey, capKey2
}

func SetupThreeMultiStoreForUnitTest() (sdk.MultiStore, *sdk.KVStoreKey, *sdk.KVStoreKey, *sdk.KVStoreKey) {
	_, ms, capKey, capKey2, capKey3 := SetupMultiStoreWithDBForUnitTest()
	return ms, capKey, capKey2, capKey3
}

func SetupMultiStoreWithDBForUnitTest() (dbm.DB, sdk.MultiStore, *sdk.KVStoreKey, *sdk.KVStoreKey, *sdk.KVStoreKey) {
	db := dbm.NewMemDB()
	capKey := sdk.NewKVStoreKey("capkey")
	capKey2 := sdk.NewKVStoreKey("capkey2")
	capKey3 := sdk.NewKVStoreKey("capkey3")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(capKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(capKey2, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(capKey3, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(common.PairStoreKey, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return db, ms, capKey, capKey2, capKey3
}

// coins to more than cover the fee
func NewNativeTokens(amount int64) sdk.Coins {
	return sdk.Coins{
		sdk.NewCoin(types.NativeTokenSymbol, amount),
	}
}

// generate a priv key and return it with its address
func PrivAndAddr() (crypto.PrivKey, sdk.AccAddress) {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())
	return priv, addr
}

func NewAccount(ctx sdk.Context, am auth.AccountKeeper, free int64) (crypto.PrivKey, sdk.Account) {
	privKey, addr := PrivAndAddr()
	acc := am.NewAccountWithAddress(ctx, addr)
	acc.SetCoins(NewNativeTokens(free))
	am.SetAccount(ctx, acc)
	return privKey, acc
}

func NewNamedAccount(ctx sdk.Context, am auth.AccountKeeper, free int64) (crypto.PrivKey, types.NamedAccount) {
	privKey, addr := PrivAndAddr()
	acc := am.NewAccountWithAddress(ctx, addr)
	acc.SetCoins(NewNativeTokens(free))

	baseAcc := auth.BaseAccount{
		Address:       acc.GetAddress(),
		Coins:         acc.GetCoins(),
		PubKey:        acc.GetPubKey(),
		AccountNumber: acc.GetAccountNumber(),
		Sequence:      acc.GetSequence(),
	}
	appAcc := &types.AppAccount{
		BaseAccount: baseAcc,
		Name:        "",
		Flags:       0x0,
	}
	am.SetAccount(ctx, appAcc)
	return privKey, appAcc
}

func NewAccountForPub(ctx sdk.Context, am auth.AccountKeeper, free, locked, freeze int64) (crypto.PrivKey, sdk.Account) {
	privKey, addr := PrivAndAddr()
	acc := am.NewAccountWithAddress(ctx, addr)
	coins := NewNativeTokens(free)
	coins = append(coins, sdk.NewCoin("XYZ-000", free))
	acc.SetCoins(coins)

	appAcc := acc.(*types.AppAccount)
	lockedCoins := NewNativeTokens(locked)
	lockedCoins = append(lockedCoins, sdk.NewCoin("XYZ-000", locked))
	appAcc.SetLockedCoins(lockedCoins)
	freezeCoins := NewNativeTokens(freeze)
	freezeCoins = append(freezeCoins, sdk.NewCoin("XYZ-000", freeze))
	appAcc.SetFrozenCoins(freezeCoins)
	am.SetAccount(ctx, acc)
	return privKey, acc
}
