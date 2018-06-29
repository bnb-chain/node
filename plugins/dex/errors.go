package dex

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 6

	// CodeIncorrectDexOperation module reserves error 400-499 lawl
	CodeIncorrectDexOperation sdk.CodeType = 400
)

// ErrIncorrectDexOperation - Error returned upon an incorrect guess
func ErrIncorrectDexOperation(codespace sdk.CodespaceType, answer string) sdk.Error {
	return sdk.NewError(codespace, CodeIncorrectDexOperation, fmt.Sprintf("Invalid dex operation: %v", answer))
}
