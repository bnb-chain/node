package timelock

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 7

	CodeInvalidDescription         sdk.CodeType = 1
	CodeInvalidLockTime            sdk.CodeType = 2
	CodeInvalidRelock              sdk.CodeType = 3
	CodeInvalidTimeLockId          sdk.CodeType = 4
	CodeTimeLockRecordDoesNotExist sdk.CodeType = 5
	CodeInvalidLockAmount          sdk.CodeType = 6
	CodeCanNotUnlock               sdk.CodeType = 7
)

//----------------------------------------
// Error constructors

func ErrInvalidDescription(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDescription, fmt.Sprintf("Invalid descrption: %s", msg))
}

func ErrInvalidLockTime(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidLockTime, fmt.Sprintf("Invalid lock time: %s", msg))
}

func ErrInvalidRelock(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidRelock, fmt.Sprintf("Invalid relock: %s", msg))
}

func ErrInvalidTimeLockId(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidTimeLockId, fmt.Sprintf("Invalid time lock id: %s", msg))
}

func ErrTimeLockRecordDoesNotExist(codespace sdk.CodespaceType, addr sdk.AccAddress, id int64) sdk.Error {
	return sdk.NewError(codespace, CodeTimeLockRecordDoesNotExist,
		fmt.Sprintf("Time lock does not exist, address=%s, id=%d", addr.String(), id))
}

func ErrInvalidLockAmount(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidLockAmount, fmt.Sprintf("Invalid lock amount: %s", msg))
}

func ErrCanNotUnlock(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeCanNotUnlock, fmt.Sprintf("Can not unlock: %s", msg))
}
