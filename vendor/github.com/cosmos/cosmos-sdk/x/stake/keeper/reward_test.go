package keeper

import (
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestAllocateReward(t *testing.T) {

	simDels := make([]types.Sharer, 11)
	// set 5% shares for the previous 10 delegator,
	// set 50% shares for the last delegator
	lastDelegator := CreateTestAddr()
	for i := 0; i < 11; i++ {
		delAddr := CreateTestAddr()
		shares := sdk.NewDecWithoutFra(5)
		if i == 10 {
			delAddr = lastDelegator
			shares = sdk.NewDecWithoutFra(50)
		}
		simDel := types.Sharer{AccAddr: delAddr, Shares: shares}
		simDels[i] = simDel
	}

	commission := sdk.NewDec(10)

	/**
	 * case1:
	 *  commission: 10
	 *  delegator1-10: 5%, delegator11: 50%
	 * expected:
	 *  delegator11, got $5
	 *  delegator1-10, 5 of them got 1, another 5 of them got 0
	 */
	rewards := allocate(simDels, commission)
	require.Len(t, rewards, 11)
	got1Count := 0
	got0Count := 0
	for _, sc := range rewards {
		if sc.AccAddr.Equals(lastDelegator) {
			require.EqualValues(t, 5, sc.Amount)
		}
		if sc.Amount == 1 {
			got1Count++
		}
		if sc.Amount == 0 {
			got0Count++
		}
	}
	require.EqualValues(t, 5, got1Count)
	require.EqualValues(t, 5, got0Count)

	/**
	 * case2:
	 *  commission: 21
	 *  delegator1-10: 5%, delegator11: 50%
	 * expected:
	 *  delegator11, got 11
	 *  delegator1-10, got 1
	 */
	commission = sdk.NewDec(21)
	rewards = allocate(simDels, commission)
	for _, sc := range rewards {
		if sc.AccAddr.Equals(lastDelegator) {
			require.EqualValues(t, 11, sc.Amount)
		} else {
			require.EqualValues(t, 1, sc.Amount)
		}
	}

	/**
	 * case3:
	 *  commission: 29
	 *  delegator1-10: 5%, delegator11: 50%
	 * expected:
	 *  delegator11, got 15
	 *  delegator1-10, 4 of them got 2, 6 of them got 1
	 *
	 */
	commission = sdk.NewDec(29)
	rewards = allocate(simDels, commission)
	got1Count = 0
	got2Count := 0
	for _, sc := range rewards {
		if sc.AccAddr.Equals(lastDelegator) {
			require.EqualValues(t, 15, sc.Amount)
		}
		if sc.Amount == 1 {
			got1Count++
		}
		if sc.Amount == 2 {
			got2Count++
		}
	}
	require.EqualValues(t, 6, got1Count)
	require.EqualValues(t, 4, got2Count)
}

func TestMulDivDecWithExtraDecimal(t *testing.T) {
	// 8/7 = 1.14285714,285714...
	a := sdk.NewDec(2e8)
	b := sdk.NewDec(4e8)
	c := sdk.NewDec(7e8)
	afterRoundDown, extraDecimalValue := mulQuoDecWithExtraDecimal(a, b, c, 1)
	require.EqualValues(t, 114285714, afterRoundDown)
	require.EqualValues(t, 2, extraDecimalValue)
	afterRoundDown, extraDecimalValue = mulQuoDecWithExtraDecimal(a, b, c, 6)
	require.EqualValues(t, 114285714, afterRoundDown)
	require.EqualValues(t, 285714, extraDecimalValue)
	// 800/7 = 114.28571428,5714...
	a = sdk.NewDec(20e8)
	b = sdk.NewDec(40e8)
	c = sdk.NewDec(7e8)
	afterRoundDown, extraDecimalValue = mulQuoDecWithExtraDecimal(a, b, c, 1)
	require.EqualValues(t, 11428571428, afterRoundDown)
	require.EqualValues(t, 5, extraDecimalValue)
	afterRoundDown, extraDecimalValue = mulQuoDecWithExtraDecimal(a, b, c, 4)
	require.EqualValues(t, 11428571428, afterRoundDown)
	require.EqualValues(t, 5714, extraDecimalValue)
	// 8000/7 = 1142.85714285,714...
	a = sdk.NewDec(200e8)
	b = sdk.NewDec(40e8)
	c = sdk.NewDec(7e8)
	afterRoundDown, extraDecimalValue = mulQuoDecWithExtraDecimal(a, b, c, 1)
	require.EqualValues(t, 114285714285, afterRoundDown)
	require.EqualValues(t, 7, extraDecimalValue)
	afterRoundDown, extraDecimalValue = mulQuoDecWithExtraDecimal(a, b, c, 3)
	require.EqualValues(t, 114285714285, afterRoundDown)
	require.EqualValues(t, 714, extraDecimalValue)
}
