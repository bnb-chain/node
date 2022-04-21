package ibc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IBC errors reserve 200 ~ 299.
const (
	DefaultCodespace sdk.CodespaceType = 3

	CodeDuplicatedSequence    sdk.CodeType = 101
	CodeFeeParamMismatch      sdk.CodeType = 102
	CodeInvalidChainId        sdk.CodeType = 103
	CodeWritePackageForbidden sdk.CodeType = 104
)

func ErrDuplicatedSequence(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeDuplicatedSequence, msg)
}

func ErrFeeParamMismatch(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeFeeParamMismatch, msg)
}

func ErrInvalidChainId(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidChainId, msg)
}

func ErrWritePackageForbidden(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeWritePackageForbidden, msg)
}
