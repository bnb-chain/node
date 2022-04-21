package keeper

import (
	"math/rand"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	"github.com/stretchr/testify/require"
)

func prepare(t *testing.T) (sdk.Context, auth.AccountKeeper, Keeper, int64, []types.Validator, [][]sdk.AccAddress, []int64, int) {
	ctx, am, k := CreateTestInput(t, false, 0)
	k.addrPool = new(sdk.Pool)
	bondDenom := k.BondDenom(ctx)

	height := int64(1000)
	height2 := int64(2000)
	height3 := int64(3000)

	minDelShares := 1
	maxDelShares := 100000

	minDelNum := 10
	maxDelNum := 500

	minCollectedFee := 1
	maxCollectedFee := 10000

	validators := make([]types.Validator, 21)
	delegators := make([][]sdk.AccAddress, 21)
	rewards := make([]int64, 21)
	rand.Seed(time.Now().UnixNano())
	var totalDelNum int
	for i := 0; i < 21; i++ {
		valPubKey := PKs[i]
		valAddr := sdk.ValAddress(valPubKey.Address().Bytes())
		validator := types.NewValidator(valAddr, valPubKey, types.Description{})

		delNum := minDelNum + rand.Intn(maxDelNum-minDelNum+1)
		var totalShares int64
		simDels := make([]types.SimplifiedDelegation, delNum)
		delsForVal := make([]sdk.AccAddress, 0)
		totalDelNum += delNum
		for j := 0; j < delNum; j++ {
			delAddr := CreateTestAddr()
			if j == 0 {
				validator.FeeAddr = delAddr
			}
			shares := int64((minDelShares + rand.Intn(maxDelShares-minDelShares+1)) * 100000000)
			totalShares += shares
			simDel := types.SimplifiedDelegation{
				DelegatorAddr: delAddr,
				Shares:        sdk.NewDec(shares),
			}
			simDels[j] = simDel
			delsForVal = append(delsForVal, delAddr)
		}
		delegators[i] = delsForVal
		k.SetSimplifiedDelegations(ctx, height, validator.OperatorAddr, simDels)

		validator.DelegatorShares = sdk.NewDec(totalShares)
		validator.Tokens = sdk.NewDec(totalShares)
		validator.DistributionAddr = Addrs[499-i]
		validator, setCommErr := validator.SetInitialCommission(types.Commission{Rate: sdk.NewDecWithPrec(40, 2), MaxRate: sdk.NewDecWithPrec(90, 2)})
		require.NoError(t, setCommErr)
		validators[i] = validator

		// simulate distribute account
		distrAcc := am.NewAccountWithAddress(ctx, validator.DistributionAddr)
		randCollectedFee := int64((minCollectedFee + rand.Intn(maxCollectedFee-minCollectedFee+1)) * 100000000)
		err := distrAcc.SetCoins(sdk.Coins{sdk.NewCoin(bondDenom, randCollectedFee)})
		require.NoError(t, err)
		rewards[i] = randCollectedFee
		am.SetAccount(ctx, distrAcc)
	}
	k.SetValidatorsByHeight(ctx, height, validators)
	k.SetValidatorsByHeight(ctx, height2, make([]types.Validator, 0))
	k.SetValidatorsByHeight(ctx, height3, make([]types.Validator, 0))
	return ctx, am, k, height, validators, delegators, rewards, totalDelNum
}

func TestDistribute(t *testing.T) {
	ctx, am, k, height, validators, delegators, rewards, totalDelNum := prepare(t)
	bondDenom := k.BondDenom(ctx)

	k.Distribute(ctx, "")

	for i, validator := range validators {
		_, found := k.GetSimplifiedDelegations(ctx, height, validator.OperatorAddr)
		require.False(t, found)

		distrAcc := am.GetAccount(ctx, validator.DistributionAddr)
		balance := distrAcc.GetCoins().AmountOf(bondDenom)
		require.Equal(t, int64(0), balance)

		var amountOfAllAccount int64
		for _, delAddr := range delegators[i] {
			delAcc := am.GetAccount(ctx, delAddr)
			amountOfAllAccount += delAcc.GetCoins().AmountOf(bondDenom)
		}
		require.Equal(t, rewards[i], amountOfAllAccount)
	}

	_, found := k.GetValidatorsByHeight(ctx, height)
	require.False(t, found)

	require.EqualValues(t, len(k.addrPool.TxRelatedAddrs()), totalDelNum+21) // add 21 distribution addresses
}

func TestDistributeInBreathBlock(t *testing.T) {
	ctx, am, k, height, validators, _, rewards, totalDelNum := prepare(t)
	bondDenom := k.BondDenom(ctx)

	k.DistributeInBreathBlock(ctx, "")

	var savedRewards []types.Reward

	// verify stored batches
	batchSize := k.GetParams(ctx).RewardDistributionBatchSize
	batchCount := int64(0)
	for ; k.hasNextBatchRewards(ctx); {
		rewards, key := k.getNextBatchRewards(ctx)
		savedRewards = append(savedRewards, rewards...)
		k.removeBatchRewards(ctx, key)

		batchCount = batchCount + 1
		require.True(t, batchSize >= int64(len(rewards)))
	}
	if int64(totalDelNum)%batchSize == 0 {
		require.True(t, batchCount == int64(totalDelNum)/batchSize)
	} else {
		require.True(t, batchCount == int64(totalDelNum)/batchSize+1)
	}

	// verify validator
	for i, validator := range validators {
		_, found := k.GetSimplifiedDelegations(ctx, height, validator.OperatorAddr)
		require.False(t, found)

		valAcc := am.GetAccount(ctx, validator.FeeAddr)
		valBalance := valAcc.GetCoins().AmountOf(bondDenom)

		distAcc := am.GetAccount(ctx, validator.DistributionAddr)
		distBalance := distAcc.GetCoins().AmountOf(bondDenom)

		totalRewardDec := sdk.NewDec(rewards[i])
		require.Equal(t, totalRewardDec.Mul(validator.Commission.Rate).RawInt(), valBalance)
		require.Equal(t, rewards[i]-totalRewardDec.Mul(validator.Commission.Rate).RawInt(), distBalance)
		require.Equal(t, rewards[i], valBalance+distBalance)

		require.NotEqual(t, int64(0), distBalance)
		require.NotEqual(t, int64(0), valBalance)
	}

	// verify delegator rewards
	require.Equal(t, totalDelNum, len(savedRewards))

	valDistAddrMap := make(map[string]sdk.AccAddress)
	for _, validator := range validators {
		valDistAddrMap[validator.OperatorAddr.String()] = validator.DistributionAddr
	}

	expectedDistAddrBalanceMap := make(map[string]int64)
	for _, reward := range savedRewards {
		distAddr := valDistAddrMap[reward.ValAddr.String()]
		if value, ok := expectedDistAddrBalanceMap[distAddr.String()]; ok {
			expectedDistAddrBalanceMap[distAddr.String()] = reward.Amount + value
		} else {
			expectedDistAddrBalanceMap[distAddr.String()] = reward.Amount
		}
	}

	for i, validator := range validators {
		totalRewardDec := sdk.NewDec(rewards[i])

		// rewards amount are correctly saved in reward store
		require.Equal(t, rewards[i]-totalRewardDec.Mul(validator.Commission.Rate).RawInt(), expectedDistAddrBalanceMap[validator.DistributionAddr.String()])

		// rewards are correctly left in distribution address
		distAcc := am.GetAccount(ctx, validator.DistributionAddr)
		distBalance := distAcc.GetCoins().AmountOf(bondDenom)
		require.Equal(t, distBalance, expectedDistAddrBalanceMap[validator.DistributionAddr.String()])
	}

	_, found := k.GetValidatorsByHeight(ctx, height)
	require.False(t, found)

	require.EqualValues(t, len(k.addrPool.TxRelatedAddrs()), 21+21) // validator fee address + distribution address
}

func TestDistributeInBlock(t *testing.T) {
	ctx, am, k, _, validators, _, _, _ := prepare(t)
	bondDenom := k.BondDenom(ctx)

	k.DistributeInBreathBlock(ctx, "")

	batchCount := k.countBatchRewards(ctx)

	for i := int64(0); i < batchCount; i++ {
		rewards, _ := k.getNextBatchRewards(ctx)

		// record delegator
		delegatorBalanceMap := make(map[string]int64) // record delegator balance before
		delegatorRewardMap := make(map[string]int64)  // record reward for each delegator in this batch
		for _, reward := range rewards {
			delegatorAcc := am.GetAccount(ctx, reward.AccAddr)
			if delegatorAcc != nil {
				delegatorBalance := delegatorAcc.GetCoins().AmountOf(bondDenom)
				delegatorBalanceMap[reward.AccAddr.String()] = delegatorBalance
			} else {
				delegatorBalanceMap[reward.AccAddr.String()] = 0
			}

			if value, ok := delegatorRewardMap[reward.AccAddr.String()]; ok {
				delegatorRewardMap[reward.AccAddr.String()] = reward.Amount + value
			} else {
				delegatorRewardMap[reward.AccAddr.String()] = reward.Amount
			}
		}

		// record distribution address
		distBalanceMap := make(map[string]int64) // record distribution address balance before
		distConsumeMap := make(map[string]int64) // record distribution address will cost amount
		valDistAddrMap := make(map[string]sdk.AccAddress)
		for _, validator := range validators {
			valDistAddrMap[validator.OperatorAddr.String()] = validator.DistributionAddr
		}

		for _, reward := range rewards {
			distAddr := valDistAddrMap[reward.ValAddr.String()]
			distAcc := am.GetAccount(ctx, distAddr)
			distBalance := distAcc.GetCoins().AmountOf(bondDenom)
			distBalanceMap[distAddr.String()] = distBalance

			if value, ok := distConsumeMap[distAddr.String()]; ok {
				distConsumeMap[distAddr.String()] = reward.Amount + value
			} else {
				distConsumeMap[distAddr.String()] = reward.Amount
			}
		}

		// do distribution
		k.DistributeInBlock(ctx, "")

		// verify distribute address balance
		for dist, balance := range distBalanceMap {
			distAddr, _ := sdk.AccAddressFromBech32(dist)
			distAcc := am.GetAccount(ctx, distAddr)
			newBalance := distAcc.GetCoins().AmountOf(bondDenom)

			require.Equal(t, newBalance, balance-distConsumeMap[dist])
		}

		// verify delegator balance
		for delegator, balance := range delegatorBalanceMap {
			delegatorAddr, _ := sdk.AccAddressFromBech32(delegator)
			delegatorAcc := am.GetAccount(ctx, delegatorAddr)
			newBalance := delegatorAcc.GetCoins().AmountOf(bondDenom)

			require.Equal(t, newBalance, balance+delegatorRewardMap[delegator])
		}
	}

	// verify reward store
	require.True(t, !k.hasNextBatchRewards(ctx))
	_, found := k.getRewardValDistAddrs(ctx)
	require.True(t, !found)
}
