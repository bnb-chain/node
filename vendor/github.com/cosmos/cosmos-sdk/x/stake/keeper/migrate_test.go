package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	"github.com/stretchr/testify/require"
)

func TestMigratePowerRankKey(t *testing.T) {
	ctx, _, k := CreateTestInput(t, false, 0)

	sdk.UpgradeMgr.AddUpgradeHeight(sdk.LaunchBscUpgrade, 10)
	sdk.UpgradeMgr.SetHeight(9)

	valPubKey := PKs[0]
	valAddr := sdk.ValAddress(valPubKey.Address().Bytes())
	validator := types.NewValidator(valAddr, valPubKey, types.Description{})
	k.SetValidator(ctx, validator)
	k.SetValidatorByPowerIndex(ctx, validator)

	store := ctx.KVStore(k.storeKey)
	opAddr := store.Get(getValidatorPowerRank(validator))
	require.Equal(t, valAddr.Bytes(), opAddr)

	sdk.UpgradeMgr.SetHeight(10)
	MigratePowerRankKey(ctx, k)
	opAddr = store.Get(getValidatorPowerRank(validator))
	require.Nil(t, opAddr)
	opAddr = store.Get(getValidatorPowerRankNew(validator))
	require.Equal(t, valAddr.Bytes(), opAddr)
}
