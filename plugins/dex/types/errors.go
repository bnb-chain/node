package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 6

	// CodeIncorrectDexOperation module reserves error 400-499
	CodeIncorrectDexOperation   sdk.CodeType = 400
	CodeInvalidOrderParam       sdk.CodeType = 401
	CodeInvalidTradeSymbol      sdk.CodeType = 402
	CodeFailInsertOrder         sdk.CodeType = 403
	CodeFailCancelOrder         sdk.CodeType = 404
	CodeFailLocateOrderToCancel sdk.CodeType = 405
	CodeDuplicatedOrder         sdk.CodeType = 406
	CodeInvalidProposal         sdk.CodeType = 407
)

// ErrIncorrectDexOperation - Error returned upon an incorrect guess
func ErrIncorrectDexOperation(answer string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeIncorrectDexOperation, fmt.Sprintf("Invalid dex operation: %v", answer))
}

func ErrInvalidOrderParam(paraName string, err string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidOrderParam, fmt.Sprintf("Invalid order parameter value - %s:%s", paraName, err))
}

func ErrInvalidTradeSymbol(err string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidTradeSymbol, fmt.Sprintf("Invalid trade symbol: %s", err))
}

func ErrInvalidProposal(err string) sdk.Error {
	return sdk.NewError(DefaultCodespace, CodeInvalidProposal, fmt.Sprintf("Invalid proposal: %s", err))
}
