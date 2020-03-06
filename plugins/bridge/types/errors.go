package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 12

	CodeInvalidTransferMsg     sdk.CodeType = 1
	CodeInvalidSequence        sdk.CodeType = 2
	CodeInvalidAmount          sdk.CodeType = 3
	CodeInvalidEthereumAddress sdk.CodeType = 4
	CodeInvalidDecimal         sdk.CodeType = 5
	CodeInvalidContractAddress sdk.CodeType = 6
	CodeTokenNotBound          sdk.CodeType = 7
	CodeInvalidSymbol          sdk.CodeType = 8
	CodeInvalidExpireTime      sdk.CodeType = 9
	CodeSerializePackageFailed sdk.CodeType = 10
	CodeGetChannelIdFailed     sdk.CodeType = 11
)

//----------------------------------------
// Error constructors

func ErrInvalidTransferMsg(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidTransferMsg, fmt.Sprintf("invalid transfer msg: %s", msg))
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

func ErrInvalidDecimal(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidDecimal, msg)
}

func ErrInvalidContractAddress(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidContractAddress, msg)
}

func ErrTokenNotBound(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeTokenNotBound, msg)
}

func ErrInvalidSymbol(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidSymbol, msg)
}

func ErrInvalidExpireTime(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidExpireTime, msg)
}

func ErrSerializePackageFailed(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeSerializePackageFailed, msg)
}

func ErrGetChannelIdFailed(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeGetChannelIdFailed, msg)
}
