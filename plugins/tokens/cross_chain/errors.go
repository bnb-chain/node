package cross_chain

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	DefaultCodespace sdk.CodespaceType = 13

	CodeInvalidDecimal         sdk.CodeType = 1
	CodeInvalidContractAddress sdk.CodeType = 2
	CodeTokenNotBind           sdk.CodeType = 3
)

func ErrInvalidDecimal(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidDecimal, msg)
}

func ErrInvalidContractAddress(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidContractAddress, msg)
}

func ErrTokenNotBind(msg string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeTokenNotBind, msg)
}
