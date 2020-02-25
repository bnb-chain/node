package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 11

	// CodeIncorrectDexOperation module reserves error 1000-1100
	CodeProphecyNotFound              sdk.CodeType = 1000
	CodeMinimumConsensusNeededInvalid sdk.CodeType = 1001
	CodeNoClaims                      sdk.CodeType = 1002
	CodeInvalidIdentifier             sdk.CodeType = 1003
	CodeProphecyFinalized             sdk.CodeType = 1004
	CodeDuplicateMessage              sdk.CodeType = 1005
	CodeInvalidClaim                  sdk.CodeType = 1006
	CodeInvalidValidator              sdk.CodeType = 1006
	CodeInternalDB                    sdk.CodeType = 1006
)

func ErrProphecyNotFound() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeProphecyNotFound, fmt.Sprintf("prophecy with given id not found"))
}

func ErrMinimumConsensusNeededInvalid() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeMinimumConsensusNeededInvalid, fmt.Sprintf("minimum consensus proportion of validator staking power must be > 0 and <= 1"))
}

func ErrNoClaims() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeNoClaims, fmt.Sprintf("cannot create prophecy without initial claim"))
}

func ErrInvalidIdentifier() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidIdentifier, fmt.Sprintf("invalid identifier provided, must be a nonempty string"))
}

func ErrProphecyFinalized() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeProphecyFinalized, fmt.Sprintf("prophecy already finalized"))
}

func ErrDuplicateMessage() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeDuplicateMessage, fmt.Sprintf("already processed message from validator for this id"))
}

func ErrInvalidClaim() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidClaim, fmt.Sprintf("claim cannot be empty string"))
}
func ErrInvalidValidator() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidValidator, fmt.Sprintf("claim must be made by actively bonded validator"))
}
func ErrInternalDB() sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInternalDB, fmt.Sprintf("failed prophecy serialization/deserialization"))
}
