package slashing

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) Unjail(ctx sdk.Context, validatorAddr sdk.ValAddress) sdk.Error {

	validator := k.validatorSet.Validator(ctx, validatorAddr)
	if validator == nil {
		return ErrNoValidatorForAddress(k.Codespace)
	}

	// cannot be unjailed if no self-delegation exists
	selfDel := k.validatorSet.Delegation(ctx, sdk.AccAddress(validator.GetFeeAddr()), validatorAddr)
	if selfDel == nil {
		return ErrMissingSelfDelegation(k.Codespace)
	}

	if validator.TokensFromShares(selfDel.GetShares()).RawInt() < k.validatorSet.MinSelfDelegation(ctx) {
		return ErrSelfDelegationTooLowToUnjail(k.Codespace)
	}

	if !validator.GetJailed() {
		return ErrValidatorNotJailed(k.Codespace)
	}

	var consAddr []byte
	if validator.IsSideChainValidator() {
		consAddr = validator.GetSideChainConsAddr()
	} else {
		consAddr = validator.GetConsAddr().Bytes()
	}

	info, found := k.getValidatorSigningInfo(ctx, consAddr)
	if !found {
		return ErrNoValidatorForAddress(k.Codespace)
	}

	// cannot be unjailed until out of jail
	if ctx.BlockHeader().Time.Before(info.JailedUntil) {
		return ErrValidatorJailed(k.Codespace)
	}

	// unjail the validator
	if validator.IsSideChainValidator() {
		k.validatorSet.UnjailSideChain(ctx, consAddr)
	} else {
		k.validatorSet.Unjail(ctx, consAddr)
	}

	return nil
}
