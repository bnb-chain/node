package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 12

	CodeInvalidAmount            sdk.CodeType = 1
	CodeInvalidEthereumAddress   sdk.CodeType = 2
	CodeInvalidDecimals          sdk.CodeType = 3
	CodeInvalidContractAddress   sdk.CodeType = 4
	CodeTokenNotBound            sdk.CodeType = 5
	CodeInvalidSymbol            sdk.CodeType = 6
	CodeInvalidExpireTime        sdk.CodeType = 7
	CodeBindRequestExists        sdk.CodeType = 8
	CodeBindRequestNotExists     sdk.CodeType = 9
	CodeTokenBound               sdk.CodeType = 10
	CodeInvalidLength            sdk.CodeType = 11
	CodeFeeNotFound              sdk.CodeType = 12
	CodeInvalidClaim             sdk.CodeType = 13
	CodeDeserializePackageFailed sdk.CodeType = 14
	CodeTokenBindRelationChanged sdk.CodeType = 15
	CodeTransferInExpire         sdk.CodeType = 16
	CodeScriptsExecutionError    sdk.CodeType = 17
	CodeInvalidMirror            sdk.CodeType = 18
	CodeMirrorSymbolExists       sdk.CodeType = 19
	CodeInvalidMirrorSync        sdk.CodeType = 20
	CodeNotBoundByMirror         sdk.CodeType = 21
	CodeMirrorSyncInvalidSupply  sdk.CodeType = 22
)

//----------------------------------------
// Error constructors

func ErrInvalidAmount(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidAmount, msg)
}

func ErrInvalidEthereumAddress(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidEthereumAddress, msg)
}

func ErrInvalidDecimals(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidDecimals, msg)
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

func ErrDeserializePackageFailed(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeDeserializePackageFailed, msg)
}

func ErrBindRequestExists(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeBindRequestExists, msg)
}

func ErrBindRequestNotExists(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeBindRequestNotExists, msg)
}

func ErrTokenBound(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeTokenBound, msg)
}

func ErrInvalidLength(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidLength, msg)
}

func ErrFeeNotFound(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeFeeNotFound, msg)
}

func ErrInvalidClaim(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidClaim, msg)
}

func ErrTokenBindRelationChanged(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeTokenBindRelationChanged, msg)
}

func ErrTransferInExpire(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeTransferInExpire, msg)
}

func ErrScriptsExecutionError(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeScriptsExecutionError, msg)
}

func ErrInvalidMirror(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidMirror, msg)
}

func ErrMirrorSymbolExists(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeMirrorSymbolExists, msg)
}

func ErrInvalidMirrorSync(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidMirrorSync, msg)
}

func ErrNotBoundByMirror(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeNotBoundByMirror, msg)
}

func ErrMirrorSyncInvalidSupply(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeMirrorSyncInvalidSupply, msg)
}
