package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/require"
)

func TestWithdrawValidatorRewardsAllNoDelegator(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewDecWithoutFra(100).RawInt(), sdk.ZeroDec())
	stakeHandler := stake.NewStakeHandler(sk)
	denom := sk.GetParams(ctx).BondDenom

	//first make a validator
	msgCreateValidator := stake.NewTestMsgCreateValidator(valOpAddr1, valConsPk1, 10)
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// allocate 100 denom of fees
	feeInputs := sdk.NewDecWithoutFra(100).RawInt()
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw self-delegation reward
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawValidatorRewardsAll(ctx, valOpAddr1)
	amt := accMapper.GetAccount(ctx, valAccAddr1).GetCoins().AmountOf(denom)
	expRes := sdk.NewDecWithPrec(90, 0).Add(sdk.NewDecWithPrec(100, 0)).TruncateInt()
	require.True(t, expRes == amt)
}

func TestWithdrawValidatorRewardsAllDelegatorNoCommission(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewDecWithoutFra(100).RawInt(), sdk.ZeroDec())
	stakeHandler := stake.NewStakeHandler(sk)
	denom := sk.GetParams(ctx).BondDenom

	//first make a validator
	msgCreateValidator := stake.NewTestMsgCreateValidator(valOpAddr1, valConsPk1, 10)
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// delegate
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, 10)
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewDecWithoutFra(90).RawInt(), amt)

	// allocate 100 denom of fees
	feeInputs := sdk.NewDecWithoutFra(100).RawInt()
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw self-delegation reward
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawValidatorRewardsAll(ctx, valOpAddr1)
	amt = accMapper.GetAccount(ctx, valAccAddr1).GetCoins().AmountOf(denom)
	expRes := sdk.NewDecWithPrec(90, 0).Add(sdk.NewDecWithPrec(100, 0).Quo(sdk.NewDecWithPrec(2, 0))).TruncateInt() // 90 + 100 tokens * 10/20
	require.True(t, expRes == amt)
}

func TestWithdrawValidatorRewardsAllDelegatorWithCommission(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewDecWithoutFra(100).RawInt(), sdk.ZeroDec())
	stakeHandler := stake.NewStakeHandler(sk)
	denom := sk.GetParams(ctx).BondDenom

	//first make a validator
	commissionRate := sdk.NewDecWithPrec(1, 1)
	msgCreateValidator := stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr1, valConsPk1, sdk.NewDecWithoutFra(10).RawInt(), commissionRate)
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// delegate
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, 10)
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewDecWithoutFra(90).RawInt(), amt)

	// allocate 100 denom of fees
	feeInputs := sdk.NewDecWithoutFra(100).RawInt()
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw validator reward
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawValidatorRewardsAll(ctx, valOpAddr1)
	amt = accMapper.GetAccount(ctx, valAccAddr1).GetCoins().AmountOf(denom)
	commissionTaken := sdk.NewDecWithPrec(100, 0).Mul(commissionRate)
	afterCommission := sdk.NewDecWithPrec(100, 0).Sub(commissionTaken)
	selfDelegationReward := afterCommission.Quo(sdk.NewDecWithPrec(2, 0))
	expRes := sdk.NewDecWithPrec(90, 0).Add(commissionTaken).Add(selfDelegationReward).TruncateInt() // 90 + 100 tokens * 10/20
	require.True(t, expRes == amt)
}

func TestWithdrawValidatorRewardsAllMultipleValidator(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewDecWithoutFra(100).RawInt(), sdk.ZeroDec())
	stakeHandler := stake.NewStakeHandler(sk)
	denom := sk.GetParams(ctx).BondDenom

	//make some  validators with different commissions
	msgCreateValidator := stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr1, valConsPk1, sdk.NewDecWithoutFra(10).RawInt(), sdk.NewDecWithPrec(1, 1))
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)

	msgCreateValidator = stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr2, valConsPk2, sdk.NewDecWithoutFra(50).RawInt(), sdk.NewDecWithPrec(2, 1))
	got = stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)

	msgCreateValidator = stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr3, valConsPk3, sdk.NewDecWithoutFra(40).RawInt(), sdk.NewDecWithPrec(3, 1))
	got = stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)

	sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// allocate 1000 denom of fees
	feeInputs := sdk.NewDecWithoutFra(1000).RawInt()
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw validator reward
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawValidatorRewardsAll(ctx, valOpAddr1)
	amt := accMapper.GetAccount(ctx, valAccAddr1).GetCoins().AmountOf(denom)

	feesInNonProposer := sdk.NewDecFromInt(feeInputs).Mul(sdk.NewDecWithPrec(95, 2))
	feesInProposer := sdk.NewDecFromInt(feeInputs).Mul(sdk.NewDecWithPrec(5, 2))
	expRes := sdk.NewDecWithPrec(90, 0). // orig tokens (100 - 10)
						Add(feesInNonProposer.Quo(sdk.NewDecWithPrec(10, 0))). // validator 1 has 1/10 total power
						Add(feesInProposer).
						TruncateInt()
	require.True(t, expRes == amt)
}

func TestWithdrawValidatorRewardsAllMultipleDelegator(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewDecWithoutFra(100).RawInt(), sdk.ZeroDec())
	stakeHandler := stake.NewStakeHandler(sk)
	denom := sk.GetParams(ctx).BondDenom

	//first make a validator with 10% commission
	commissionRate := sdk.NewDecWithPrec(1, 1)
	msgCreateValidator := stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr1, valConsPk1, sdk.NewDecWithoutFra(10).RawInt(), sdk.NewDecWithPrec(1, 1))
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// delegate
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, 10)
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewDecWithoutFra(90).RawInt(), amt)

	msgDelegate = stake.NewTestMsgDelegate(delAddr2, valOpAddr1, 20)
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt = accMapper.GetAccount(ctx, delAddr2).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewDecWithoutFra(80).RawInt(), amt)

	// allocate 100 denom of fees
	feeInputs := sdk.NewDecWithoutFra(100).RawInt()
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw validator reward
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawValidatorRewardsAll(ctx, valOpAddr1)
	amt = accMapper.GetAccount(ctx, valAccAddr1).GetCoins().AmountOf(denom)

	commissionTaken := sdk.NewDecWithPrec(100, 0).Mul(commissionRate)
	afterCommission := sdk.NewDecWithPrec(100, 0).Sub(commissionTaken)
	expRes := sdk.NewDecWithPrec(90, 0).
		Add(afterCommission.Quo(sdk.NewDecWithPrec(4, 0))).
		Add(commissionTaken).
		TruncateInt() // 90 + 100*90% tokens * 10/40
	require.True(t, expRes == amt)
}
