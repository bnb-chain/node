package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestWithdrawRewards(t *testing.T) {

	// initialize
	height := int64(0)
	fp := InitialFeePool()
	vi := NewValidatorDistInfo(valAddr1, height)
	commissionRate := sdk.NewDecWithPrec(2, 2)
	validatorTokens := sdk.NewDecWithoutFra(10)
	validatorDelShares := sdk.NewDecWithoutFra(10)
	totalBondedTokens := validatorTokens.Add(sdk.NewDecWithoutFra(90)) // validator-1 is 10% of total power

	di1 := NewDelegationDistInfo(delAddr1, valAddr1, height)
	di1Shares := sdk.NewDecWithoutFra(5) // this delegator has half the shares in the validator

	di2 := NewDelegationDistInfo(delAddr2, valAddr1, height)
	di2Shares := sdk.NewDecWithoutFra(5)

	// simulate adding some stake for inflation
	height = 10
	fp.Pool = DecCoins{NewDecCoin("stake", sdk.NewDecWithoutFra(1000).RawInt())}

	// withdraw rewards
	di1, vi, fp, rewardRecv1 := di1.WithdrawRewards(fp, vi, height, totalBondedTokens,
		validatorTokens, validatorDelShares, di1Shares, commissionRate)

	assert.Equal(t, height, di1.WithdrawalHeight)
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(900), fp.TotalValAccum.Accum))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(900), fp.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(49), vi.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(2), vi.PoolCommission[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(49), rewardRecv1[0].Amount))

	// add more blocks and inflation
	height = 20
	fp.Pool[0].Amount = fp.Pool[0].Amount.Add(sdk.NewDecWithoutFra(1000))

	// withdraw rewards
	di2, vi, fp, rewardRecv2 := di2.WithdrawRewards(fp, vi, height, totalBondedTokens,
		validatorTokens, validatorDelShares, di2Shares, commissionRate)

	assert.Equal(t, height, di2.WithdrawalHeight)
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(1800), fp.TotalValAccum.Accum))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(1800), fp.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(49), vi.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(4), vi.PoolCommission[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(98), rewardRecv2[0].Amount))
}
