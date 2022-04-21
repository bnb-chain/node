// nolint
package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CodeType = sdk.CodeType

const (
	DefaultCodespace sdk.CodespaceType = 12

	CodeMissSideChainId    CodeType = 101
	CodeInvalidSideChainId CodeType = 102
	CodeInvalidCrossChainPackage CodeType = 103
)

func ErrMissSideChainId(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeMissSideChainId, "side chain id is missing")
}

func ErrInvalidSideChainId(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidSideChainId, msg)
}

func ErrInvalidCrossChainPackage(codespace sdk.CodespaceType) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidCrossChainPackage, "invalid cross chain package")
}
