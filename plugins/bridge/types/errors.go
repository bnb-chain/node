package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 12

	CodeInvalidTransferMsg     sdk.CodeType = 1
	CodeInvalidSymbol          sdk.CodeType = 2
	CodeInvalidSequence        sdk.CodeType = 3
	CodeInvalidAmount          sdk.CodeType = 4
	CodeInvalidEthereumAddress sdk.CodeType = 5
)

//----------------------------------------
// Error constructors

func ErrInvalidTransferMsg(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidTransferMsg, fmt.Sprintf("invalid transfer msg: %s", msg))
}

func ErrInvalidSymbol(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidSymbol, msg)
}

func ErrInvalidSequence(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidSequence, msg)
}

func ErrInvalidAmount(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidAmount, msg)
}

func ErrInvalidEthereumAddress(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidEthereumAddress, msg)
}
