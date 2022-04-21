// nolint
package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CodeType = sdk.CodeType

const (
	DefaultCodespace sdk.CodespaceType = 4

	CodeInvalidValidator         CodeType = 101
	CodeInvalidDelegation        CodeType = 102
	CodeInvalidInput             CodeType = 103
	CodeValidatorJailed          CodeType = 104
	CodeInvalidProposal          CodeType = 105
	CodeInvalidSideChain         CodeType = 106
	CodeInvalidCrossChainPackage CodeType = 107
	CodeInvalidAddress           CodeType = sdk.CodeInvalidAddress
	CodeUnauthorized             CodeType = sdk.CodeUnauthorized
	CodeInternal                 CodeType = sdk.CodeInternal
	CodeUnknownRequest           CodeType = sdk.CodeUnknownRequest
)

//validator
func ErrNilValidatorAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "validator address is nil")
}

func ErrNilValidatorConsAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "validator consensus address is nil")
}

func ErrBadValidatorAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidAddress, "validator address is invalid")
}

func ErrInvalidDelegator(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "delegator address is invalid")
}

func ErrNoValidatorFound(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "validator does not exist for that address")
}

func ErrValidatorOwnerExists(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "validator already exist for this operator address, must use new validator operator address")
}

func ErrValidatorPubKeyExists(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "validator already exist for this pubkey, must use new validator pubkey")
}

func ErrValidatorSideConsAddrExists(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "validator already exist for this sideConsAddr, must use new validator sideConsAddr")
}

func ErrValidatorJailed(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "validator for this address is currently jailed")
}

func ErrBadRemoveValidator(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "error removing validator")
}

func ErrEmptyMoniker(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "moniker cannot be empty")
}

func ErrDescriptionLength(codespace sdk.CodespaceType, descriptor string, got, max int) sdk.Error {
	msg := fmt.Sprintf("bad description length for %v, got length %v, max is %v", descriptor, got, max)
	return sdk.NewError(codespace, CodeInvalidValidator, msg)
}

func ErrCommissionNegative(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "commission must be positive")
}

func ErrCommissionHuge(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "commission cannot be more than 100%")
}

func ErrCommissionGTMaxRate(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "commission cannot be more than the max rate")
}

func ErrCommissionUpdateTime(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "commission cannot be changed more than once in 24h")
}

func ErrCommissionChangeRateNegative(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "commission change rate must be positive")
}

func ErrCommissionChangeRateGTMaxRate(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "commission change rate cannot be more than the max rate")
}

func ErrCommissionGTMaxChangeRate(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "commission cannot be changed more than max change rate")
}

func ErrNilDelegatorAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "delegator address is nil")
}

func ErrNilLauncherAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "launcher address of remove validator is nil")
}

func ErrBadDenom(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "invalid coin denomination")
}

func ErrBadDelegationAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "unexpected address length for this (address, validator) pair")
}

func ErrBadDelegationAmount(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "invalid amount: "+msg)
}

func ErrNoDelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "no delegation for this (address, validator) pair")
}

func ErrBadDelegatorAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "delegator does not exist for that address")
}

func ErrNoDelegatorForAddress(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "delegator does not contain this delegation")
}

func ErrInsufficientShares(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "insufficient delegation shares")
}

func ErrDelegationValidatorEmpty(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "cannot delegate to an empty validator")
}

func ErrNotEnoughDelegationAmount(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "not enough delegation amount")
}

func ErrNotEnoughDelegationShares(codespace sdk.CodespaceType, shares string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, fmt.Sprintf("not enough shares only have %v", shares))
}

func ErrBadSharesAmount(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "shares must be > 0")
}

func ErrBadSharesPercent(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "shares percent must be >0 and <=1")
}

func ErrNotMature(codespace sdk.CodespaceType, operation, descriptor string, got, min time.Time) sdk.Error {
	msg := fmt.Sprintf("%v is not mature requires a min %v of %v, currently it is %v",
		operation, descriptor, got, min)
	return sdk.NewError(codespace, CodeUnauthorized, msg)
}

func ErrNoUnbondingDelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "no unbonding delegation found")
}

func ErrExistingUnbondingDelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "existing unbonding delegation found")
}

func ErrBadRedelegationAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "unexpected address length for this (address, srcValidator, dstValidator) tuple")
}

func ErrNoRedelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "no redelegation found")
}

func ErrBadRedelegationSrc(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "redelegation src validator not found")
}

func ErrBadRedelegationDst(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "redelegation dst validator not found")
}

func ErrSelfRedelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "cannot redelegate to the same validator")
}

func ErrInvalidRedelegator(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation, "self-delegator cannot redelegate to other validators")
}

func ErrTransitiveRedelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation,
		"redelegation to this validator already in progress, first redelegation to this validator must complete before next redelegation")
}

func ErrConflictingRedelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDelegation,
		"conflicting redelegation from this source validator to this dest validator already exists, you must wait for it to finish")
}

func ErrBothShareMsgsGiven(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "both shares amount and shares percent provided")
}

func ErrNeitherShareMsgsGiven(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, "neither shares amount nor shares percent provided")
}

func ErrMissingSignature(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "missing signature")
}

func ErrInvalidProposal(codespace sdk.CodespaceType, reason string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidProposal, "invalid proposal: %s", reason)
}

func ErrInvalidSideChainId(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidSideChain, "invalid side chain id")
}

func ErrInvalidCrosschainPackage(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidCrossChainPackage, "invalid cross chain package")
}
