//nolint
package slashing

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Local code type
type CodeType = sdk.CodeType

const (
	// Default slashing codespace
	DefaultCodespace sdk.CodespaceType = 10

	CodeInvalidInput                 CodeType = 100
	CodeInvalidValidator             CodeType = 101
	CodeValidatorJailed              CodeType = 102
	CodeValidatorNotJailed           CodeType = 103
	CodeMissingSelfDelegation        CodeType = 104
	CodeSelfDelegationTooLowToUnjail CodeType = 105
	CodeInvalidClaim                 CodeType = 106

	CodeExpiredEvidence        CodeType = 201
	CodeFailSlash              CodeType = 202
	CodeHandledEvidence        CodeType = 203
	CodeInvalidEvidence        CodeType = 204
	CodeInvalidSideChain       CodeType = 205
	CodeDuplicateDowntimeClaim CodeType = 206
)

func ErrNoValidatorForAddress(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "that address is not associated with any known validator")
}

func ErrBadValidatorAddr(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidValidator, "validator does not exist for that address")
}

func ErrValidatorJailed(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeValidatorJailed, "validator still jailed, cannot yet be unjailed")
}

func ErrValidatorNotJailed(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeValidatorNotJailed, "validator not jailed, cannot be unjailed")
}

func ErrMissingSelfDelegation(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeMissingSelfDelegation, "validator has no self-delegation; cannot be unjailed")
}

func ErrSelfDelegationTooLowToUnjail(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeSelfDelegationTooLowToUnjail, "validator's self delegation less than minimum; cannot be unjailed")
}

func ErrInvalidClaim(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidClaim, msg)
}

func ErrExpiredEvidence(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeExpiredEvidence, "The given evidences are expired")
}

func ErrFailedToSlash(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeFailSlash, fmt.Sprintf("failed to slash, %s", msg))
}

func ErrEvidenceHasBeenHandled(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeHandledEvidence, "The evidence has been handled")
}

func ErrInvalidEvidence(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidEvidence, msg)
}

func ErrInvalidSideChainId(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidSideChain, "invalid side chain id")
}

func ErrDuplicateDowntimeClaim(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeDuplicateDowntimeClaim, "duplicate downtime claim")
}

func ErrInvalidInput(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidInput, msg)
}
