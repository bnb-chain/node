package swap

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 8

	CodeInvalidOtherChainAddress   sdk.CodeType = 1
	CodeInvalidRandomNumberHash    sdk.CodeType = 2
	CodeInvalidRandomNumber        sdk.CodeType = 3
	CodeInvalidSwapOutAmount       sdk.CodeType = 4
	CodeInvalidHeightSpan          sdk.CodeType = 5
	CodeDuplicatedRandomNumberHash sdk.CodeType = 6
	CodeClaimExpiredSwap           sdk.CodeType = 7
	CodeRefundUnexpiredSwap        sdk.CodeType = 8
	CodeMismatchedRandomNumber     sdk.CodeType = 9
	CodeNonExistRandomNumberHash   sdk.CodeType = 10
	CodeTooLargeQueryLimit         sdk.CodeType = 11
	CodeUnexpectedSwapStatus       sdk.CodeType = 12
	CodeInvalidTimestamp           sdk.CodeType = 13
	CodeInvalidResponseSwap        sdk.CodeType = 14
	CodeResponseExpiredSwap        sdk.CodeType = 15
)

func ErrInvalidOtherChainAddress(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidOtherChainAddress, msg)
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

func ErrDuplicatedRandomNumberHash(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeDuplicatedRandomNumberHash, msg)
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

func ErrNonExistRandomNumberHash(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeNonExistRandomNumberHash, msg)
}

func ErrTooLargeQueryLimit(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeTooLargeQueryLimit, msg)
}

func ErrUnexpectedSwapStatus(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeUnexpectedSwapStatus, msg)
}

func ErrCodeInvalidTimestamp(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidTimestamp, msg)
}

func ErrCodeInvalidResponseSwap(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidResponseSwap, msg)
}

func ErrCodeResponseExpiredSwap(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeResponseExpiredSwap, msg)
}
