package utils

import (
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	dbm "github.com/tendermint/tendermint/libs/db"
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
