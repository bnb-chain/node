package sidechain

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 31

	CodeInvalidSideChainId sdk.CodeType = 101
)

func ErrInvalidSideChainId(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidSideChainId, msg)
}
