package app

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/bnb-chain/node/app/pub"
	"github.com/bnb-chain/node/common/log"
)

func NewValAddrCache(stakeKeeper stake.Keeper) *ValAddrCache {
	cache := &ValAddrCache{
		cache:       make(map[string]sdk.AccAddress),
		stakeKeeper: stakeKeeper,
	}

	return cache
}

type ValAddrCache struct {
	cache                 map[string]sdk.AccAddress
	distributionAddrCache map[string]sdk.AccAddress
	stakeKeeper           stake.Keeper
}

func (vac *ValAddrCache) ClearCache() {
	vac.cache = make(map[string]sdk.AccAddress)
	vac.distributionAddrCache = make(map[string]sdk.AccAddress)
}

func (vac *ValAddrCache) SetAccAddr(consAddr sdk.ConsAddress, accAddr sdk.AccAddress) {
	vac.cache[string(consAddr)] = accAddr
}

func (vac *ValAddrCache) GetAccAddr(ctx sdk.Context, consAddr sdk.ConsAddress) sdk.AccAddress {
	if value, ok := vac.cache[string(consAddr)]; ok {
		return value
	}
	validator, found := vac.stakeKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if !found {
		panic(fmt.Errorf("can't load validator with consensus address %s", consAddr.String()))
	}
	accAddr := validator.GetFeeAddr()
	vac.SetAccAddr(consAddr, accAddr)
	return accAddr
}

func (vac *ValAddrCache) SetDistributionAddr(consAddr sdk.ConsAddress, accAddr sdk.AccAddress) {
	vac.distributionAddrCache[string(consAddr)] = accAddr
}

func (vac *ValAddrCache) GetDistributionAddr(ctx sdk.Context, consAddr sdk.ConsAddress) sdk.AccAddress {
	if value, ok := vac.distributionAddrCache[string(consAddr)]; ok {
		return value
	}
	validator, found := vac.stakeKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if !found {
		panic(fmt.Errorf("can't load validator with consensus address %s", consAddr.String()))
	}
	distributionAddr := validator.DistributionAddr
	vac.SetDistributionAddr(consAddr, distributionAddr)
	return distributionAddr
}

func distributeFeeBEPHHH(ctx sdk.Context, am auth.AccountKeeper, valAddrCache *ValAddrCache, publishBlockFee bool, stakeKeeper stake.Keeper) (blockFee pub.BlockFee) {
	// TODO: needs to add BSC ratio Fee
	fee := fees.Pool.BlockFees()
	blockFee = pub.BlockFee{Height: ctx.BlockHeader().Height}
	if fee.IsEmpty() {
		// no fees in this block
		return
	}

	proposerValAddr := ctx.BlockHeader().ProposerAddress
	proposerDistributionAddr := valAddrCache.GetDistributionAddr(ctx, proposerValAddr)

	// distrubute proposer rewards
	proposerRewards := sdk.Coins{}
	feeForAllRewards := sdk.Coins{}
	var baseProposerRewardRatio int64 = 1  // 1%
	var bonusProposerRewardRatio int64 = 4 // 4%
	voteNum := int64(len(ctx.VoteInfos()))
	// TODO: ensure it's the right way to get current validators
	currentValidators := stakeKeeper.GetLastValidators(ctx)
	validatorNum := int64(len(currentValidators))
	for _, token := range fee.Tokens {
		amount := token.Amount
		proposerAmount := (amount*baseProposerRewardRatio + amount*bonusProposerRewardRatio*voteNum/validatorNum) / 100
		proposerRewards = append(proposerRewards, sdk.NewCoin(token.Denom, proposerAmount))
		feeForAllRewards = append(feeForAllRewards, sdk.NewCoin(token.Denom, amount-proposerAmount))
	}
	proposerDistributionAcc := am.GetAccount(ctx, proposerDistributionAddr)
	_ = proposerDistributionAcc.SetCoins(proposerDistributionAcc.GetCoins().Plus(proposerRewards))
	am.SetAccount(ctx, proposerDistributionAcc)
	feeForAllAcc := am.GetAccount(ctx, stake.FeeForAllAccAddr)
	_ = feeForAllAcc.SetCoins(feeForAllAcc.GetCoins().Plus(feeForAllRewards))
	am.SetAccount(ctx, feeForAllAcc)

	//if publishBlockFee {
	//	blockFee.Fee = fee.String()
	//	for _, validator := range currentValidators {
	//		blockFee.Validators = append(blockFee.Validators, validator.ConsAddress().String())
	//	}
	//}
	return
}

func distributeFee(ctx sdk.Context, am auth.AccountKeeper, valAddrCache *ValAddrCache, publishBlockFee bool) (blockFee pub.BlockFee) {
	fee := fees.Pool.BlockFees()
	blockFee = pub.BlockFee{Height: ctx.BlockHeader().Height}
	if fee.IsEmpty() {
		// no fees in this block
		return
	}

	proposerValAddr := ctx.BlockHeader().ProposerAddress
	proposerAccAddr := valAddrCache.GetAccAddr(ctx, proposerValAddr)
	voteInfos := ctx.VoteInfos()
	valSize := int64(len(voteInfos))
	var validators []string
	if publishBlockFee {
		validators = make([]string, 0, valSize)
		validators = append(validators, string(proposerAccAddr)) // the first validator to publish should be proposer
	}

	if fee.Type == sdk.FeeForProposer {
		// The proposer's account must be initialized before it becomes a proposer.
		proposerAcc := am.GetAccount(ctx, proposerAccAddr)
		_ = proposerAcc.SetCoins(proposerAcc.GetCoins().Plus(fee.Tokens))
		am.SetAccount(ctx, proposerAcc)
	} else if fee.Type == sdk.FeeForAll {
		log.Info("Distributing the fees to all the validators",
			"totalFees", fee.Tokens, "validatorSize", valSize)
		avgTokens := sdk.Coins{}
		roundingTokens := sdk.Coins{}
		for _, token := range fee.Tokens {
			amount := token.Amount
			avgAmount := amount / valSize
			roundingAmount := amount - avgAmount*valSize
			if avgAmount != 0 {
				avgTokens = append(avgTokens, sdk.NewCoin(token.Denom, avgAmount))
			}

			if roundingAmount != 0 {
				roundingTokens = append(roundingTokens, sdk.NewCoin(token.Denom, roundingAmount))
			}
		}

		if avgTokens.IsZero() {
			proposerAcc := am.GetAccount(ctx, proposerAccAddr)
			_ = proposerAcc.SetCoins(proposerAcc.GetCoins().Plus(fee.Tokens))
			am.SetAccount(ctx, proposerAcc)
		} else {
			for _, voteInfo := range voteInfos {
				validator := voteInfo.Validator
				accAddr := valAddrCache.GetAccAddr(ctx, validator.Address)
				validatorAcc := am.GetAccount(ctx, accAddr)
				if bytes.Equal(proposerValAddr, validator.Address) {
					if !roundingTokens.IsZero() {
						_ = validatorAcc.SetCoins(validatorAcc.GetCoins().Plus(roundingTokens))
					}
				} else if publishBlockFee {
					validators = append(validators, string(accAddr))
				}
				_ = validatorAcc.SetCoins(validatorAcc.GetCoins().Plus(avgTokens))
				am.SetAccount(ctx, validatorAcc)
			}
		}
	}

	if publishBlockFee {
		blockFee.Fee = fee.String()
		blockFee.Validators = validators
	}

	return
}
