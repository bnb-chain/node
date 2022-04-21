package sidechain

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/require"
)

func CreateTestInput(t *testing.T, isCheckTx bool) (sdk.Context, Keeper) {
	key := sdk.NewKVStoreKey("sc")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(key, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	cdc := codec.New()
	paramsKeeper := params.NewKeeper(cdc, keyParams, tkeyParams)

	mode := sdk.RunTxModeDeliver
	if isCheckTx {
		mode = sdk.RunTxModeCheck
	}
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "foochainid"}, mode, log.NewNopLogger())
	k := NewKeeper(key, paramsKeeper.Subspace(DefaultParamspace), cdc)
	k.SetParams(ctx, DefaultParams())
	return ctx, k
}

func TestKeeper_SetSideChainIdAndStorePrefix(t *testing.T) {
	ctx, keeper := CreateTestInput(t, false)

	scIds, scPrefixes := keeper.GetAllSideChainPrefixes(ctx)
	require.Equal(t, len(scIds), 0)
	require.Equal(t, len(scPrefixes), 0)

	keeper.SetSideChainIdAndStorePrefix(ctx, "abc", []byte{0x11, 0x12})
	keeper.SetSideChainIdAndStorePrefix(ctx, "xyz", []byte{0xab})
	scIds, scPrefixes = keeper.GetAllSideChainPrefixes(ctx)
	require.Equal(t, len(scIds), 2)
	require.Equal(t, len(scPrefixes), 2)
	require.Equal(t, scIds[0], "abc")
	require.Equal(t, scPrefixes[0], []byte{0x11, 0x12})
	require.Equal(t, scIds[1], "xyz")
	require.Equal(t, scPrefixes[1], []byte{0xab})
}
