package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/stake/types"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
)

func TestSetParams(t *testing.T) {
	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(tkeyStake, sdk.StoreTypeTransient, nil)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	cdc := MakeTestCodec()
	mode := sdk.RunTxModeDeliver
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "foochainid"}, mode, log.NewNopLogger())
	pk := params.NewKeeper(cdc, keyParams, tkeyParams)
	k := NewKeeper(cdc, keyStake, keyStakeReward, tkeyStake, nil, nil, pk.Subspace(DefaultParamspace), types.DefaultCodespace)
	sdk.UpgradeMgr.AddUpgradeHeight(sdk.LaunchBscUpgrade, 10)
	sdk.UpgradeMgr.AddUpgradeHeight(sdk.BEP128, 100)

	sdk.UpgradeMgr.SetHeight(1)
	k.SetParams(ctx, types.DefaultParams())
	require.True(t, k.paramstore.Has(ctx, types.KeyUnbondingTime))
	require.True(t, k.paramstore.Has(ctx, types.KeyMaxValidators))
	require.True(t, k.paramstore.Has(ctx, types.KeyBondDenom))
	require.False(t, k.paramstore.Has(ctx, types.KeyMinSelfDelegation))
	require.False(t, k.paramstore.Has(ctx, types.KeyMinDelegationChange))

	sdk.UpgradeMgr.SetHeight(10)
	k.SetParams(ctx, types.DefaultParams())
	require.True(t, k.paramstore.Has(ctx, types.KeyUnbondingTime))
	require.True(t, k.paramstore.Has(ctx, types.KeyMaxValidators))
	require.True(t, k.paramstore.Has(ctx, types.KeyBondDenom))
	require.True(t, k.paramstore.Has(ctx, types.KeyMinSelfDelegation))
	require.True(t, k.paramstore.Has(ctx, types.KeyMinDelegationChange))

	sdk.UpgradeMgr.SetHeight(100)
	k.SetParams(ctx, types.DefaultParams())
	require.True(t, k.paramstore.Has(ctx, types.KeyUnbondingTime))
	require.True(t, k.paramstore.Has(ctx, types.KeyMaxValidators))
	require.True(t, k.paramstore.Has(ctx, types.KeyBondDenom))
	require.True(t, k.paramstore.Has(ctx, types.KeyMinSelfDelegation))
	require.True(t, k.paramstore.Has(ctx, types.KeyMinDelegationChange))
	require.True(t, k.paramstore.Has(ctx, types.KeyRewardDistributionBatchSize))
}
