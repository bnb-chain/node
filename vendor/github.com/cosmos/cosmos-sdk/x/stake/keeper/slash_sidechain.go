package keeper

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

func (k Keeper) SlashSideChain(ctx sdk.Context, sideChainId string, sideConsAddr []byte, slashAmount sdk.Dec) (sdk.Validator, sdk.Dec, error) {
	logger := ctx.Logger().With("module", "x/stake")

	sideCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, sideChainId)
	if err != nil {
		return nil, sdk.ZeroDec(), errors.New("invalid side chain id")
	}

	validator, found := k.GetValidatorBySideConsAddr(sideCtx, sideConsAddr)
	if !found {
		// If not found, the validator must have been overslashed and removed - so we don't need to do anything
		// NOTE:  Correctness dependent on invariant that unbonding delegations / redelegations must also have been completely
		//        slashed in this case - which we don't explicitly check, but should be true.
		// Log the slash attempt for future reference (maybe we should tag it too)
		logger.Error(fmt.Sprintf(
			"WARNING: Ignored attempt to slash a nonexistent validator with address %s, we recommend you investigate immediately",
			sdk.HexEncode(sideConsAddr)))
		return nil, sdk.ZeroDec(), nil
	}

	// should not be slashing unbonded
	if validator.IsUnbonded() {
		return nil, sdk.ZeroDec(), errors.New(fmt.Sprintf("should not be slashing unbonded validator: %s", validator.GetOperator()))
	}

	if !validator.Jailed {
		k.JailSideChain(sideCtx, sideConsAddr)
	}

	selfDelegation, found := k.GetDelegation(sideCtx, validator.FeeAddr, validator.OperatorAddr)
	remainingSlashAmount := slashAmount
	if found {
		slashShares := validator.SharesFromTokens(slashAmount)
		slashSelfDelegationShares := sdk.MinDec(slashShares, selfDelegation.Shares)
		if slashSelfDelegationShares.RawInt() > 0 {
			unbondAmount, err := k.unbond(sideCtx, selfDelegation.DelegatorAddr, validator.OperatorAddr, slashSelfDelegationShares)
			if err != nil {
				return nil, sdk.ZeroDec(), errors.New(fmt.Sprintf("error unbonding delegator: %v", err))
			}
			remainingSlashAmount = remainingSlashAmount.Sub(unbondAmount)
		}
	}

	if remainingSlashAmount.RawInt() > 0 {
		ubd, found := k.GetUnbondingDelegation(sideCtx, validator.FeeAddr, validator.OperatorAddr)
		if found {
			slashUnBondingAmount := sdk.MinInt64(remainingSlashAmount.RawInt(), ubd.Balance.Amount)
			ubd.Balance.Amount = ubd.Balance.Amount - slashUnBondingAmount
			k.SetUnbondingDelegation(sideCtx, ubd)
			remainingSlashAmount = remainingSlashAmount.Sub(sdk.NewDec(slashUnBondingAmount))
		}
	}

	slashedAmt := slashAmount.Sub(remainingSlashAmount)

	bondDenom := k.BondDenom(ctx)
	delegationAccBalance := k.bankKeeper.GetCoins(ctx, DelegationAccAddr)
	slashedCoin := sdk.NewCoin(bondDenom, slashedAmt.RawInt())
	if err := k.bankKeeper.SetCoins(ctx, DelegationAccAddr, delegationAccBalance.Minus(sdk.Coins{slashedCoin})); err != nil {
		return nil, slashedAmt, err
	}
	if ctx.IsDeliverTx() && k.addrPool != nil {
		k.addrPool.AddAddrs([]sdk.AccAddress{DelegationAccAddr})
	}
	if validator.IsBonded() {
		ibcPackage := types.IbcValidatorSetPackage{
			Type: types.JailPackageType,
			ValidatorSet: []types.IbcValidator{
				{
					ConsAddr: validator.SideConsAddr,
					FeeAddr:  validator.SideFeeAddr,
					DistAddr: validator.DistributionAddr,
					Power:    uint64(validator.GetPower().RawInt()),
				},
			},
		}
		if _, err := k.SaveValidatorSetToIbc(ctx, sideChainId, ibcPackage); err != nil {
			return nil, sdk.ZeroDec(), errors.New(err.Error())
		}
	}

	return validator, slashedAmt, nil

}

// jail a validator
func (k Keeper) JailSideChain(ctx sdk.Context, consAddr []byte) {
	validator := k.mustGetValidatorBySideConsAddr(ctx, consAddr)
	k.jailValidator(ctx, validator)
	k.Logger(ctx).Info(fmt.Sprintf("validator %s jailed", hex.EncodeToString(consAddr)))
	// TODO Return event(s), blocked on https://github.com/tendermint/tendermint/pull/1803
	return
}

// unjail a validator
func (k Keeper) UnjailSideChain(ctx sdk.Context, consAddr []byte) {
	validator := k.mustGetValidatorBySideConsAddr(ctx, consAddr)
	k.unjailValidator(ctx, validator)
	k.Logger(ctx).Info(fmt.Sprintf("validator %s unjailed", hex.EncodeToString(consAddr)))
	// TODO Return event(s), blocked on https://github.com/tendermint/tendermint/pull/1803
	return
}

// return sharers as a array about delegation shares and amount receiving address for each validator,
// and totalShares as total shares added by all these validators
func convertValidators2Shares(validators []types.Validator) (sharers []types.Sharer) {
	sharers = make([]types.Sharer, len(validators))
	for i, val := range validators {
		sharers[i] = types.Sharer{AccAddr: val.DistributionAddr, Shares: val.DelegatorShares}
	}
	return sharers
}

// return this map for storing data of validators amount receiving detail. the receiving address as map key, and amount as map value
func (k Keeper) AllocateSlashAmtToValidators(ctx sdk.Context, slashedConsAddr []byte, amount sdk.Dec) (bool, map[string]int64, error) {
	// allocate remaining rewards to validators who are going to be distributed next time.
	validators, found := k.GetEarliestValidatorsWithHeight(ctx)
	if !found {
		return found, nil, nil
	}
	// remove bad validator if it exists in the eligible validators
	for i := 0; i < len(validators); i++ {
		if bytes.Compare(validators[i].SideConsAddr, slashedConsAddr) == 0 {
			if i == len(validators)-1 {
				validators = validators[:i]
			} else {
				validators = append(validators[:i], validators[i+1:]...)
			}
			break
		}
	}

	if len(validators) == 0 {
		return false, nil, nil
	}

	bondDenom := k.BondDenom(ctx)
	sharers := convertValidators2Shares(validators)
	rewards := allocate(sharers, amount)

	validatorsCompensation := make(map[string]int64)
	changedAddrs := make([]sdk.AccAddress, len(rewards))
	for i := range rewards {
		accBalance := k.bankKeeper.GetCoins(ctx, rewards[i].AccAddr)
		rewardCoin := sdk.Coins{sdk.NewCoin(bondDenom, rewards[i].Amount)}
		accBalance.Plus(rewardCoin)
		if err := k.bankKeeper.SetCoins(ctx, rewards[i].AccAddr, accBalance.Plus(rewardCoin)); err != nil {
			return found, validatorsCompensation, err
		}
		changedAddrs[i] = rewards[i].AccAddr
		validatorsCompensation[string(rewards[i].AccAddr.Bytes())] = rewards[i].Amount
	}
	if ctx.IsDeliverTx() && k.addrPool != nil {
		k.addrPool.AddAddrs(changedAddrs)
	}
	return found, validatorsCompensation, nil
}
