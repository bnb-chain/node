package slashing

import (
	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

func TestSideChainSlashDowntime(t *testing.T) {

	slashingParams := DefaultParams()
	slashingParams.MaxEvidenceAge = 12 * 60 * 60 * time.Second
	ctx, sideCtx, _, stakeKeeper, _, keeper := createSideTestInput(t, slashingParams)

	// create a validator
	bondAmount := int64(10000e8)
	realSlashAmt := sdk.MinInt64(slashingParams.DowntimeSlashAmount, bondAmount)
	valAddr := addrs[0]
	sideConsAddr, sideFeeAddr := createSideAddr(20), createSideAddr(20)
	msgCreateVal := newTestMsgCreateSideValidator(valAddr, sideConsAddr, sideFeeAddr, bondAmount)
	got := stake.NewHandler(stakeKeeper, gov.Keeper{})(ctx, msgCreateVal)
	require.True(t, got.IsOK(), "expected create validator msg to be ok, got: %v", got)
	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)

	sideHeight := uint64(100)
	sideChainId := "bsc"
	sideTimestamp := ctx.BlockHeader().Time.Add(-6 * 60 * 60 * time.Second)
	claim := SideDowntimeSlashPackage{
		SideConsAddr:  sideConsAddr,
		SideHeight:    sideHeight,
		SideChainId:   sdk.ChainID(1),
		SideTimestamp: uint64(sideTimestamp.Unix()),
	}

	result := keeper.slashingSideDowntime(ctx, &claim)

	require.Nil(t, result, "Expected nil, but got : %v", result)

	info, found := keeper.getValidatorSigningInfo(sideCtx, sideConsAddr)
	require.True(t, found)
	require.EqualValues(t, ctx.BlockHeader().Time.Add(slashingParams.DowntimeUnbondDuration).Unix(), info.JailedUntil.Unix())

	slashRecord, found := keeper.getSlashRecord(sideCtx, sideConsAddr, Downtime, sideHeight)
	require.True(t, found)
	require.EqualValues(t, sideHeight, slashRecord.InfractionHeight)
	require.EqualValues(t, sideChainId, slashRecord.SideChainId)
	require.EqualValues(t, realSlashAmt, slashRecord.SlashAmt)
	require.EqualValues(t, ctx.BlockHeader().Time.Add(slashingParams.DowntimeUnbondDuration).Unix(), slashRecord.JailUntil.Unix())

	validator, found := stakeKeeper.GetValidatorBySideConsAddr(sideCtx, sideConsAddr)
	require.True(t, found)
	require.True(t, validator.Jailed)
	require.EqualValues(t, bondAmount-realSlashAmt, validator.Tokens.RawInt())
	require.EqualValues(t, bondAmount-realSlashAmt, validator.DelegatorShares.RawInt())

	delegation, found := stakeKeeper.GetDelegation(sideCtx, validator.FeeAddr, validator.OperatorAddr)
	require.True(t, found)
	require.EqualValues(t, bondAmount-realSlashAmt, delegation.Shares.RawInt())

	result = keeper.slashingSideDowntime(ctx, &claim)
	require.NotNil(t, result)
	require.EqualValues(t, CodeDuplicateDowntimeClaim, result.Code())

	exeResult := keeper.ExecuteSynPackage(ctx, []byte(""), 0)
	require.NotNil(t, exeResult.Err)

	claim.SideHeight = 0
	bz, _ := rlp.EncodeToBytes(&claim)
	_, result = keeper.checkSideDowntimeSlashPackage(bz)
	require.NotNil(t, result)

	claim.SideHeight = sideHeight
	claim.SideConsAddr = createSideAddr(21)

	result = keeper.slashingSideDowntime(ctx, &claim)
	require.NotNil(t, result)

	claim.SideConsAddr = sideConsAddr
	claim.SideTimestamp = uint64(ctx.BlockHeader().Time.Add(-24 * 60 * 60 * time.Second).Unix())
	result = keeper.slashingSideDowntime(ctx, &claim)
	require.EqualValues(t, CodeExpiredEvidence, result.Code(), "Expected got 201 err code, but got err: %v", result)

	claim.SideTimestamp = uint64(ctx.BlockHeader().Time.Add(-6 * 60 * 60 * time.Second).Unix())
	claim.SideConsAddr = sideConsAddr
	claim.SideChainId = sdk.ChainID(2)

	result = keeper.slashingSideDowntime(ctx, &claim)
	require.NotNil(t, result, "Expected get err, but got nil")
	require.EqualValues(t, CodeInvalidSideChain, result.Code(), "Expected got 205 error code, but got err: %v", result)

	claim.SideHeight = sideHeight
	claim.SideConsAddr = createSideAddr(20)
	claim.SideChainId = sdk.ChainID(1)

	result = keeper.slashingSideDowntime(ctx, &claim)
	require.NotNil(t, result, "Expected got err of no signing info found, but got nil")

}

func TestSlashDowntimeBalanceVerify(t *testing.T) {

	slashingParams := DefaultParams()
	slashingParams.MaxEvidenceAge = 12 * 60 * 60 * time.Second
	slashingParams.DowntimeSlashAmount = 8000e8
	slashingParams.DowntimeSlashFee = 5000e8
	ctx, sideCtx, bk, stakeKeeper, _, keeper := createSideTestInput(t, slashingParams)

	bondAmount := int64(10000e8)
	// create validator to be allocated slashed amount further
	valAddr1 := addrs[0]
	sideConsAddr1, sideFeeAddr1 := createSideAddr(20), createSideAddr(20)
	msgCreateVal := newTestMsgCreateSideValidator(valAddr1, sideConsAddr1, sideFeeAddr1, bondAmount)
	ctx = ctx.WithBlockHeight(100)
	got := stake.NewHandler(stakeKeeper, gov.Keeper{})(ctx, msgCreateVal)
	require.True(t, got.IsOK(), "expected create validator msg to be ok, got: %v", got)
	validator1, found := stakeKeeper.GetValidator(sideCtx, valAddr1)
	require.True(t, found)
	distributionAddr := validator1.DistributionAddr
	stake.EndBreatheBlock(ctx, stakeKeeper)

	// create a validator will be slashed amount
	ctx = ctx.WithBlockHeight(200)
	valAddr2 := addrs[1]
	sideConsAddr2, sideFeeAddr2 := createSideAddr(20), createSideAddr(20)
	msgCreateVal = newTestMsgCreateSideValidator(valAddr2, sideConsAddr2, sideFeeAddr2, bondAmount)
	got = stake.NewHandler(stakeKeeper, gov.Keeper{})(ctx, msgCreateVal)
	require.True(t, got.IsOK(), "expected create validator msg to be ok, got: %v", got)
	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)

	sideHeight := uint64(50)
	sideTimestamp := ctx.BlockHeader().Time.Add(-6 * 60 * 60 * time.Second)
	claim := SideDowntimeSlashPackage{
		SideConsAddr:  sideConsAddr2,
		SideHeight:    sideHeight,
		SideChainId:   sdk.ChainID(1),
		SideTimestamp: uint64(sideTimestamp.Unix()),
	}

	feesInPoolBefore := fees.Pool.BlockFees().Tokens.AmountOf("steak")
	result := keeper.slashingSideDowntime(ctx, &claim)
	require.Nil(t, result)

	validator2, found := stakeKeeper.GetValidator(sideCtx, valAddr2)
	require.True(t, found)
	require.True(t, validator2.Jailed)
	require.EqualValues(t, 2000e8, validator2.Tokens.RawInt())
	require.EqualValues(t, 2000e8, validator2.DelegatorShares.RawInt())

	delegation, found := stakeKeeper.GetDelegation(sideCtx, sdk.AccAddress(valAddr2), valAddr2)
	require.True(t, found)
	require.EqualValues(t, 2000e8, delegation.Shares.RawInt()) // slashed 8000e8 from validator2 delegation

	require.EqualValues(t, 5000e8, fees.Pool.BlockFees().Tokens.AmountOf("steak")-feesInPoolBefore) // add 5000e8 as DowntimeSlashFee to fee pool

	coins := bk.GetCoins(ctx, distributionAddr)
	require.EqualValues(t, 3000e8, coins.AmountOf("steak")) // remaining amount(3000e8) allocated to

	sideHeight = uint64(80)
	sideTimestamp = ctx.BlockHeader().Time.Add(-3 * 60 * 60 * time.Second)
	claim = SideDowntimeSlashPackage{
		SideConsAddr:  sideConsAddr2,
		SideHeight:    sideHeight,
		SideChainId:   sdk.ChainID(1),
		SideTimestamp: uint64(sideTimestamp.Unix()),
	}

	result = keeper.slashingSideDowntime(ctx, &claim)
	require.Nil(t, result)

	validator2, found = stakeKeeper.GetValidator(sideCtx, valAddr2)
	require.True(t, found)
	require.True(t, validator2.Jailed)
	require.EqualValues(t, 0, validator2.Tokens.RawInt())
	require.EqualValues(t, 0, validator2.DelegatorShares.RawInt())

	_, found = stakeKeeper.GetDelegation(sideCtx, sdk.AccAddress(valAddr2), valAddr2)
	require.False(t, found)

	realSlashedAmount := int64(2000e8)
	require.EqualValues(t, slashingParams.DowntimeSlashFee+realSlashedAmount, fees.Pool.BlockFees().Tokens.AmountOf("steak")-feesInPoolBefore)

	coins = bk.GetCoins(ctx, distributionAddr)
	require.EqualValues(t, 3000e8, coins.AmountOf("steak"))

	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)
	_, found = stakeKeeper.GetValidator(sideCtx, valAddr2)
	require.False(t, found)
}
