package testutils

import (
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	dbm "github.com/tendermint/tendermint/libs/db"

	"github.com/BiJie/BinanceChain/common/account"

	"github.com/BiJie/BinanceChain/common/types"
)

func SetupMultiStoreForUnitTest() (sdk.MultiStore, *sdk.KVStoreKey, *sdk.KVStoreKey) {
	_, ms, capKey, capKey2 := SetupMultiStoreWithDBForUnitTest()
	return ms, capKey, capKey2
}

func SetupMultiStoreWithDBForUnitTest() (dbm.DB, sdk.MultiStore, *sdk.KVStoreKey, *sdk.KVStoreKey) {
	db := dbm.NewMemDB()
	capKey := sdk.NewKVStoreKey("capkey")
	capKey2 := sdk.NewKVStoreKey("capkey2")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(capKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(capKey2, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return db, ms, capKey, capKey2
}

// coins to more than cover the fee
func NewNativeTokens(amount int64) sdk.Coins {
	return sdk.Coins{
		sdk.NewInt64Coin(types.NativeToken, amount),
	}
}

// generate a priv key and return it with its address
func PrivAndAddr() (crypto.PrivKey, sdk.AccAddress) {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())
	return priv, addr
}

func NewAccount(ctx types.Context, am account.Mapper, free int64) (crypto.PrivKey, auth.Account) {
	privKey, addr := PrivAndAddr()
	acc := am.NewAccountWithAddress(ctx, addr)
	acc.SetCoins(NewNativeTokens(free))
	am.SetAccount(ctx, acc)
	return privKey, acc
}
