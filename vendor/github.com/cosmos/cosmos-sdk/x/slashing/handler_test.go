package slashing

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCannotUnjailUnlessJailed(t *testing.T) {
	// initial setup
	ctx, ck, sk, _, keeper := createTestInput(t, DefaultParams())
	slh := NewSlashingHandler(keeper)
	amtInt := sdk.NewDecWithoutFra(10000).RawInt()
	addr, val, amt := addrs[0], pks[0], amtInt
	msg := NewTestMsgCreateValidator(addr, val, amt)
	got := stake.NewStakeHandler(sk)(ctx, msg)
	fmt.Println(got.Log)
	require.True(t, got.IsOK())
	stake.EndBlocker(ctx, sk)
	require.Equal(t, ck.GetCoins(ctx, sdk.AccAddress(addr)), sdk.Coins{{sk.GetParams(ctx).BondDenom, initCoins - amt}})
	require.True(t, sdk.NewDecFromInt(amt).Equal(sk.Validator(ctx, addr).GetPower()))

	// assert non-jailed validator can't be unjailed
	got = slh(ctx, NewMsgUnjail(addr))
	require.False(t, got.IsOK(), "allowed unjail of non-jailed validator")
	require.Equal(t, sdk.ToABCICode(DefaultCodespace, CodeValidatorNotJailed), got.Code)
}

func TestJailedValidatorDelegations(t *testing.T) {
	slashParams := DefaultParams()

	ctx, _, stakeKeeper, _, slashingKeeper := createTestInput(t, slashParams)

	stakeParams := stakeKeeper.GetParams(ctx)
	stakeParams.UnbondingTime = 0
	stakeKeeper.SetParams(ctx, stakeParams)

	// create a validator
	amount := int64(stakeParams.MinSelfDelegation)
	valPubKey, bondAmount := pks[0], amount
	valAddr, _ := addrs[1], sdk.ConsAddress(addrs[0])

	msgCreateVal := NewTestMsgCreateValidator(valAddr, valPubKey, sdk.NewDec(bondAmount).RawInt())
	got := stake.NewStakeHandler(stakeKeeper)(ctx, msgCreateVal)
	require.True(t, got.IsOK(), "expected create validator msg to be ok, got: %v", got)

	// end block
	stake.EndBlocker(ctx, stakeKeeper)

	// delegate tokens to the validator
	delAddr := sdk.AccAddress(addrs[2])
	msgDelegate := newTestMsgDelegate(delAddr, valAddr, sdk.NewDec(bondAmount).RawInt())
	got = stake.NewStakeHandler(stakeKeeper)(ctx, msgDelegate)
	require.True(t, got.IsOK(), "expected delegation to be ok, got %v", got)

	unbondShares := sdk.NewDec(sdk.NewDecWithoutFra(10).RawInt())

	// unbond validator total self-delegations (which should jail the validator)
	msgBeginUnbonding := stake.NewMsgBeginUnbonding(sdk.AccAddress(valAddr), valAddr, unbondShares)
	got = stake.NewStakeHandler(stakeKeeper)(ctx, msgBeginUnbonding)
	require.True(t, got.IsOK(), "expected begin unbonding validator msg to be ok, got: %v", got)

	_, err := stakeKeeper.CompleteUnbonding(ctx, sdk.AccAddress(valAddr), valAddr)
	require.Nil(t, err, "expected complete unbonding validator to be ok, got: %v", err)

	// verify validator still exists and is jailed
	validator, found := stakeKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)
	require.True(t, validator.GetJailed())

	// verify the validator cannot unjail itself
	got = NewSlashingHandler(slashingKeeper)(ctx, NewMsgUnjail(valAddr))
	require.False(t, got.IsOK(), "expected jailed validator to not be able to unjail, got: %v", got)

	// self-delegate to validator
	msgSelfDelegate := newTestMsgDelegate(sdk.AccAddress(valAddr), valAddr, sdk.NewDec(bondAmount).RawInt())
	got = stake.NewStakeHandler(stakeKeeper)(ctx, msgSelfDelegate)
	require.True(t, got.IsOK(), "expected delegation to not be ok, got %v", got)

	// verify the validator cannot unjail itself
	got = NewSlashingHandler(slashingKeeper)(ctx, NewMsgUnjail(valAddr))
	require.False(t, got.IsOK(), "expected jailed validator to not be able to unjail, got: %v", got)

	ctx = ctx.WithBlockTime(ctx.BlockHeader().Time.Add(slashParams.TooLowDelUnbondDuration))
	// verify the validator can now unjail itself
	got = NewSlashingHandler(slashingKeeper)(ctx, NewMsgUnjail(valAddr))
	require.True(t, got.IsOK(), "expected jailed validator to be able to unjail, got: %v", got)
}
