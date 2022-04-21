package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

const (
	// for getting the snapshot of validators in the specific days ago
	daysBackwardForValidatorSnapshot = 3
	// the count of blocks to distribute a day's rewards should be smaller than this value
	boundOfRewardDistributionBlockCount = int64(10000)
)

func (k Keeper) Distribute(ctx sdk.Context, sideChainId string) {

	// The rewards collected yesterday is decided by the validators the day before yesterday.
	// So this distribution is for the validators bonded 2 days ago
	validators, height, found := k.GetHeightValidatorsByIndex(ctx, daysBackwardForValidatorSnapshot)
	// be noted: if len(validators) == 0, it still needs to call removeValidatorsAndDelegationsAtHeight
	if !found { // do nothing, if there is no validators to be rewarded.
		return
	}

	bondDenom := k.BondDenom(ctx)
	var toPublish []types.DistributionData
	for _, validator := range validators {
		distAccCoins := k.bankKeeper.GetCoins(ctx, validator.DistributionAddr)
		totalReward := distAccCoins.AmountOf(bondDenom)
		totalRewardDec := sdk.ZeroDec()
		commission := sdk.ZeroDec()
		rewards := make([]types.PreReward, 0)
		if totalReward > 0 {
			delegations, found := k.GetSimplifiedDelegations(ctx, height, validator.OperatorAddr)
			if !found {
				panic(fmt.Sprintf("no delegations found with height=%d, validator=%s", height, validator.OperatorAddr))
			}
			totalRewardDec = sdk.NewDec(totalReward)
			commission = totalRewardDec.Mul(validator.Commission.Rate)
			remainReward := totalRewardDec.Sub(commission)
			// remove all balance of bondDenom from Distribution account
			distAccCoins = distAccCoins.Minus(sdk.Coins{sdk.NewCoin(bondDenom, totalReward)})
			if err := k.bankKeeper.SetCoins(ctx, validator.DistributionAddr, distAccCoins); err != nil {
				panic(err)
			}
			rewards = allocate(simDelsToSharers(delegations), remainReward)
			if commission.RawInt() > 0 { // assign rewards to self-delegator
				if _, _, err := k.bankKeeper.AddCoins(ctx, validator.GetFeeAddr(), sdk.Coins{sdk.NewCoin(bondDenom, commission.RawInt())}); err != nil {
					panic(err)
				}
			}
			// assign rewards to delegator
			changedAddrs := make([]sdk.AccAddress, len(rewards)+1)
			for i := range rewards {
				if _, _, err := k.bankKeeper.AddCoins(ctx, rewards[i].AccAddr, sdk.Coins{sdk.NewCoin(bondDenom, rewards[i].Amount)}); err != nil {
					panic(err)
				}
				changedAddrs[i] = rewards[i].AccAddr
			}

			changedAddrs[len(rewards)] = validator.DistributionAddr
			if k.addrPool != nil {
				k.addrPool.AddAddrs(changedAddrs)
			}
		}

		if ctx.IsDeliverTx() && k.PbsbServer != nil {
			var toPublishRewards []types.Reward
			for i := range rewards {
				tokens, err := sdk.MulQuoDec(validator.GetTokens(), rewards[i].Shares, validator.GetDelegatorShares())
				if err != nil {
					panic(err)
				}
				toPublishReward := types.Reward{
					ValAddr: validator.GetOperator(),
					AccAddr: rewards[i].AccAddr,
					Tokens:  tokens,
					Amount:  rewards[i].Amount,
				}
				toPublishRewards = append(toPublishRewards, toPublishReward)
			}

			toPublish = append(toPublish, types.DistributionData{
				Validator:      validator.GetOperator(),
				SelfDelegator:  validator.GetFeeAddr(),
				DistributeAddr: validator.DistributionAddr,
				ValShares:      validator.GetDelegatorShares(),
				ValTokens:      validator.GetTokens(),
				TotalReward:    totalRewardDec,
				Commission:     commission,
				Rewards:        toPublishRewards,
			})

		}
	}

	if ctx.IsDeliverTx() && len(toPublish) > 0 && k.PbsbServer != nil {
		event := types.SideDistributionEvent{
			SideChainId: sideChainId,
			Data:        toPublish,
		}
		k.PbsbServer.Publish(event)
	}

	removeValidatorsAndDelegationsAtHeight(height, k, ctx, validators)
}

// DistributeInBreathBlock will 1) calculate rewards as Distribute does, 2) transfer commissions to all validators, and
// 3) save delegator's rewards to reward store for later distribution.
func (k Keeper) DistributeInBreathBlock(ctx sdk.Context, sideChainId string) {
	// if there are left reward distribution batches in the previous day, will distribute all of them here
	// this is only a safe guard to make sure that all the previous day's rewards are distributed
	// because this case should happen in very very special case (e.g., bc maintenance for a long time), so there is no much optimization here
	for ; k.hasNextBatchRewards(ctx); {
		k.distributeSingleBatch(ctx, sideChainId)
	}

	validators, height, found := k.GetHeightValidatorsByIndex(ctx, daysBackwardForValidatorSnapshot)
	if !found {
		return
	}

	var toPublish []types.DistributionData           // data to be published in breathe blocks
	var toSaveRewards []types.Reward                 // rewards to be saved
	var toSaveValDistAddrs []types.StoredValDistAddr // mapping between validator and distribution address, to be saved

	bondDenom := k.BondDenom(ctx)
	for _, validator := range validators {
		distAccCoins := k.bankKeeper.GetCoins(ctx, validator.DistributionAddr)
		totalReward := distAccCoins.AmountOf(bondDenom)
		totalRewardDec := sdk.ZeroDec()
		commission := sdk.ZeroDec()
		rewards := make([]types.PreReward, 0)
		if totalReward > 0 {
			delegations, found := k.GetSimplifiedDelegations(ctx, height, validator.OperatorAddr)
			if !found {
				panic(fmt.Sprintf("no delegations found with height=%d, validator=%s", height, validator.OperatorAddr))
			}
			totalRewardDec = sdk.NewDec(totalReward)

			//distribute commission
			commission = totalRewardDec.Mul(validator.Commission.Rate)
			if commission.RawInt() > 0 {
				if _, _, err := k.bankKeeper.AddCoins(ctx, validator.GetFeeAddr(), sdk.Coins{sdk.NewCoin(bondDenom, commission.RawInt())}); err != nil {
					panic(err)
				}
				if _, _, err := k.bankKeeper.SubtractCoins(ctx, validator.DistributionAddr, sdk.Coins{sdk.NewCoin(bondDenom, commission.RawInt())}); err != nil {
					panic(err)
				}
			}

			//calculate rewards for delegators
			remainReward := totalRewardDec.Sub(commission)
			rewards = allocate(simDelsToSharers(delegations), remainReward)
			for i := range rewards {
				// previous tokens calculation is in `node` repo, move it to here
				tokens, err := sdk.MulQuoDec(validator.GetTokens(), rewards[i].Shares, validator.GetDelegatorShares())
				if err != nil {
					panic(err)
				}
				toSaveReward := types.Reward{
					ValAddr: validator.GetOperator(),
					AccAddr: rewards[i].AccAddr,
					Tokens:  tokens,
					Amount:  rewards[i].Amount,
				}
				toSaveRewards = append(toSaveRewards, toSaveReward)
			}

			//track validator and distribution address mapping
			toSaveValDistAddrs = append(toSaveValDistAddrs, types.StoredValDistAddr{
				Validator:      validator.OperatorAddr,
				DistributeAddr: validator.DistributionAddr})

			//update address pool
			changedAddrs := [2]sdk.AccAddress{validator.FeeAddr, validator.DistributionAddr}
			if k.addrPool != nil {
				k.addrPool.AddAddrs(changedAddrs[:])
			}
		}

		if ctx.IsDeliverTx() && k.PbsbServer != nil {
			toPublish = append(toPublish, types.DistributionData{
				Validator:      validator.GetOperator(),
				SelfDelegator:  validator.GetFeeAddr(),
				DistributeAddr: validator.DistributionAddr,
				ValShares:      validator.GetDelegatorShares(),
				ValTokens:      validator.GetTokens(),
				TotalReward:    totalRewardDec,
				Commission:     commission,
				Rewards:        nil, //do not publish rewards in breathe blocks
			})
		}
	}

	if len(toSaveRewards) > 0 { //to save rewards
		//1) get batch size from parameters, 2) hard limit to make sure rewards can be distributed in a day
		batchSize := getDistributionBatchSize(k.GetParams(ctx).RewardDistributionBatchSize, int64(len(toSaveRewards)))
		batchCount := int64(len(toSaveRewards)) / batchSize
		if int64(len(toSaveRewards))%batchSize != 0 {
			batchCount = batchCount + 1
		}

		// save rewards
		var batchNo = int64(0)
		for ; batchNo < batchCount-1; batchNo++ {
			k.setBatchRewards(ctx, batchNo, toSaveRewards[batchNo*batchSize:(batchNo+1)*batchSize])
		}
		k.setBatchRewards(ctx, batchNo, toSaveRewards[batchNo*batchSize:])

		// save validator <-> distribution address map
		k.setRewardValDistAddrs(ctx, toSaveValDistAddrs)
	}

	// publish data if needed
	if ctx.IsDeliverTx() && len(toPublish) > 0 && k.PbsbServer != nil {
		event := types.SideDistributionEvent{
			SideChainId: sideChainId,
			Data:        toPublish,
		}
		k.PbsbServer.Publish(event)
	}

	removeValidatorsAndDelegationsAtHeight(height, k, ctx, validators)
}

// DistributeInBlock will 1) actually distribute rewards to delegators, using reward store, 2) clear reward store if needed
func (k Keeper) DistributeInBlock(ctx sdk.Context, sideChainId string) {
	if hasNext := k.hasNextBatchRewards(ctx); !hasNext { // already done the distribution of rewards
		return
	}

	k.distributeSingleBatch(ctx, sideChainId)
}

// distributeSingleBatch will distribute an single batch of rewards if there is any
func (k Keeper) distributeSingleBatch(ctx sdk.Context, sideChainId string) {
	// get batch rewards and validator <-> distribution address mapping
	rewards, key := k.getNextBatchRewards(ctx)
	valDistAddrs, found := k.getRewardValDistAddrs(ctx)
	if !found {
		panic("cannot find required mapping")
	}

	valDistAddrMap := make(map[string]sdk.AccAddress)
	for _, valDist := range valDistAddrs {
		valDistAddrMap[valDist.Validator.String()] = valDist.DistributeAddr
	}

	distAddrBalanceMap := make(map[string]int64) // track distribute address balance changes
	var toPublish []types.DistributionData       // data to be published in blocks
	var toPublishRewards []types.Reward          // rewards to be published in blocks

	var changedAddrs []sdk.AccAddress //changed addresses

	bondDenom := k.BondDenom(ctx)
	for _, reward := range rewards {
		distAddr := valDistAddrMap[reward.ValAddr.String()]
		if value, ok := distAddrBalanceMap[distAddr.String()]; ok {
			distAddrBalanceMap[distAddr.String()] = reward.Amount + value
		} else {
			distAddrBalanceMap[distAddr.String()] = reward.Amount
		}

		if _, _, err := k.bankKeeper.AddCoins(ctx, reward.AccAddr, sdk.Coins{sdk.NewCoin(bondDenom, reward.Amount)}); err != nil {
			panic(err)
		}

		toPublishRewards = append(toPublishRewards, reward)
		changedAddrs = append(changedAddrs, reward.AccAddr)
	}

	for addr, value := range distAddrBalanceMap {
		accAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			panic(err)
		}
		if _, _, err := k.bankKeeper.SubtractCoins(ctx, accAddr, sdk.Coins{sdk.NewCoin(bondDenom, value)}); err != nil {
			panic(err)
		}
		changedAddrs = append(changedAddrs, accAddr)
	}

	// delete the batch in store
	k.removeBatchRewards(ctx, key)

	// check whether this batch is the last one
	if hasNext := k.hasNextBatchRewards(ctx); !hasNext {
		k.removeRewardValDistAddrs(ctx)
	}

	//update address pool
	if k.addrPool != nil {
		k.addrPool.AddAddrs(changedAddrs[:])
	}

	// publish data if needed
	if ctx.IsDeliverTx() && len(toPublishRewards) > 0 && k.PbsbServer != nil {
		toPublish = append(toPublish, types.DistributionData{
			Validator:      nil,
			SelfDelegator:  nil,
			DistributeAddr: nil,
			ValShares:      sdk.Dec{},
			ValTokens:      sdk.Dec{},
			TotalReward:    sdk.Dec{},
			Commission:     sdk.Dec{},
			Rewards:        toPublishRewards, // only publish rewards in normal block
		})
		event := types.SideDistributionEvent{
			SideChainId: sideChainId,
			Data:        toPublish,
		}
		k.PbsbServer.Publish(event)
	}
}

// getDistributionBatchSize will adjust batch size to make sure all rewards will be distribute in a day (pre-defined block number)
// usually the batch size will not be changed, just for prevention
func getDistributionBatchSize(batchSize, totalRewardLen int64) int64 {
	if totalRewardLen/boundOfRewardDistributionBlockCount >= batchSize {
		batchSize = totalRewardLen / (boundOfRewardDistributionBlockCount / 2)
	}
	return batchSize
}

func simDelsToSharers(simDels []types.SimplifiedDelegation) []types.Sharer {
	sharers := make([]types.Sharer, len(simDels))
	for i, del := range simDels {
		sharers[i] = types.Sharer{AccAddr: del.DelegatorAddr, Shares: del.Shares}
	}
	return sharers
}

func removeValidatorsAndDelegationsAtHeight(height int64, k Keeper, ctx sdk.Context, validators []types.Validator) {
	for _, validator := range validators {
		k.RemoveSimplifiedDelegations(ctx, height, validator.OperatorAddr)
	}
	k.RemoveValidatorsByHeight(ctx, height)
}
