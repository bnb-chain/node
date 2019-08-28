package swap

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 8

	CodeInvalidAddrOtherChain          sdk.CodeType = 1
	CodeInvalidRandomNumberHash        sdk.CodeType = 2
	CodeInvalidRandomNumber            sdk.CodeType = 3
	CodeInvalidSwapOutAmount           sdk.CodeType = 4
	CodeInvalidHeightSpan              sdk.CodeType = 5
	CodeDuplicatedSwapID               sdk.CodeType = 6
	CodeClaimExpiredSwap               sdk.CodeType = 7
	CodeRefundUnexpiredSwap            sdk.CodeType = 8
	CodeMismatchedRandomNumber         sdk.CodeType = 9
	CodeNonExistSwapID                 sdk.CodeType = 10
	CodeTooLargeQueryLimit             sdk.CodeType = 11
	CodeUnexpectedSwapStatus           sdk.CodeType = 12
	CodeInvalidTimestamp               sdk.CodeType = 13
	CodeInvalidSingleChainSwap         sdk.CodeType = 14
	CodeExpectedIncomeTooLong          sdk.CodeType = 15
	CodeUnexpectedClaimSingleChainSwap sdk.CodeType = 16
)

func ErrInvalidAddrOtherChain(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidAddrOtherChain, msg)
}

func ErrInvalidRandomNumberHash(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidRandomNumberHash, msg)
}

func ErrInvalidRandomNumber(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidRandomNumber, msg)
}

func ErrInvalidSwapOutAmount(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidSwapOutAmount, msg)
}

func ErrInvalidHeightSpan(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidHeightSpan, msg)
}

func ErrDuplicatedSwapID(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeDuplicatedSwapID, msg)
}

func ErrClaimExpiredSwap(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeClaimExpiredSwap, msg)
}

func ErrRefundUnexpiredSwap(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeRefundUnexpiredSwap, msg)
}

func ErrMismatchedRandomNumber(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeMismatchedRandomNumber, msg)
}

func ErrNonExistSwapID(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeNonExistSwapID, msg)
}

func ErrTooLargeQueryLimit(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeTooLargeQueryLimit, msg)
}

func ErrUnexpectedSwapStatus(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeUnexpectedSwapStatus, msg)
}

func ErrInvalidTimestamp(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidTimestamp, msg)
}

func ErrInvalidSingleChainSwap(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidSingleChainSwap, msg)
}

func ErrExpectedIncomeTooLong(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeExpectedIncomeTooLong, msg)
}

func ErrUnexpectedClaimSingleChainSwap(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeUnexpectedClaimSingleChainSwap, msg)
}
