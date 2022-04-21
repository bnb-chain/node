package slashing

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) onValidatorBonded(ctx sdk.Context, address sdk.ConsAddress, _ sdk.ValAddress) {
	// Update the signing info start height or create a new signing info
	_, found := k.getValidatorSigningInfo(ctx, address)
	if !found {
		signingInfo := ValidatorSigningInfo{
			StartHeight:         ctx.BlockHeight(),
			IndexOffset:         0,
			JailedUntil:         time.Unix(0, 0),
			MissedBlocksCounter: 0,
		}
		k.setValidatorSigningInfo(ctx, address, signingInfo)
	}

	// Create a new slashing period when a validator is bonded
	slashingPeriod := ValidatorSlashingPeriod{
		ValidatorAddr: address,
		StartHeight:   ctx.BlockHeight(),
		EndHeight:     0,
		SlashedSoFar:  sdk.ZeroDec(),
	}
	k.addOrUpdateValidatorSlashingPeriod(ctx, slashingPeriod)
}

func (k Keeper) onSideChainValidatorBonded(ctx sdk.Context, sideConsAddr []byte, _ sdk.ValAddress) {
	// Update the signing info start height or create a new signing info
	_, found := k.getValidatorSigningInfo(ctx, sideConsAddr)
	if !found {
		signingInfo := ValidatorSigningInfo{
			StartHeight:         ctx.BlockHeight(),
			IndexOffset:         0,
			JailedUntil:         time.Unix(0, 0),
			MissedBlocksCounter: 0,
		}
		k.setValidatorSigningInfo(ctx, sideConsAddr, signingInfo)
	}
}

// Mark the slashing period as having ended when a validator begins unbonding
func (k Keeper) onValidatorBeginUnbonding(ctx sdk.Context, address sdk.ConsAddress, _ sdk.ValAddress) {
	slashingPeriod := k.getValidatorSlashingPeriodForHeight(ctx, address, ctx.BlockHeight())
	slashingPeriod.EndHeight = ctx.BlockHeight()
	k.addOrUpdateValidatorSlashingPeriod(ctx, slashingPeriod)
}

// Create SigningInfo and jail the validator
func (k Keeper) onSelfDelDropBelowMin(ctx sdk.Context, valAddress sdk.ValAddress) {
	validator := k.validatorSet.Validator(ctx, valAddress)
	if validator == nil {
		return
	}
	var consAddr []byte
	if validator.IsSideChainValidator() {
		consAddr = validator.GetSideChainConsAddr()
	} else {
		consAddr = validator.GetConsAddr().Bytes()
	}

	header := ctx.BlockHeader()
	signingInfo, found := k.getValidatorSigningInfo(ctx, consAddr)
	if !found {
		signingInfo := ValidatorSigningInfo{
			StartHeight:         header.Height,
			IndexOffset:         0,
			JailedUntil:         header.Time.Add(k.TooLowDelUnbondDuration(ctx)),
			MissedBlocksCounter: 0,
		}
		k.setValidatorSigningInfo(ctx, consAddr, signingInfo)
	} else {
		signingInfo.JailedUntil = header.Time.Add(k.TooLowDelUnbondDuration(ctx))
		k.setValidatorSigningInfo(ctx, consAddr, signingInfo)
	}
}

//_________________________________________________________________________________________

// Wrapper struct
type Hooks struct {
	k Keeper
}

var _ sdk.StakingHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// Implements sdk.ValidatorHooks
func (h Hooks) OnValidatorBonded(ctx sdk.Context, address sdk.ConsAddress, operator sdk.ValAddress) {
	h.k.onValidatorBonded(ctx, address, operator)
}

// Implements sdk.ValidatorHooks
func (h Hooks) OnSideChainValidatorBonded(ctx sdk.Context, sideConsAddr []byte, operator sdk.ValAddress) {
	h.k.onSideChainValidatorBonded(ctx, sideConsAddr, operator)
}

// Implements sdk.ValidatorHooks
func (h Hooks) OnValidatorBeginUnbonding(ctx sdk.Context, address sdk.ConsAddress, operator sdk.ValAddress) {
	h.k.onValidatorBeginUnbonding(ctx, address, operator)
}

// Implements sdk.ValidatorHooks
func (h Hooks) OnSelfDelDropBelowMin(ctx sdk.Context, operator sdk.ValAddress) {
	h.k.onSelfDelDropBelowMin(ctx, operator)
}

// nolint - unused hooks
func (h Hooks) OnValidatorCreated(_ sdk.Context, _ sdk.ValAddress)                           {}
func (h Hooks) OnValidatorModified(_ sdk.Context, _ sdk.ValAddress)                          {}
func (h Hooks) OnValidatorRemoved(_ sdk.Context, _ sdk.ValAddress)                           {}
func (h Hooks) OnDelegationCreated(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress)        {}
func (h Hooks) OnDelegationSharesModified(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) {}
func (h Hooks) OnDelegationRemoved(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress)        {}
func (h Hooks) OnSideChainValidatorBeginUnbonding(ctx sdk.Context, sideConsAddr []byte, operator sdk.ValAddress) {
}
