package timelock

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MsgRoute = "timelock"

	MaxDescriptionLength = 128
)

var _ sdk.Msg = TimeLockMsg{}

type TimeLockMsg struct {
	From        sdk.AccAddress `json:"from"`
	Description string         `json:"description"`
	Amount      sdk.Coins      `json:"amount"`
	LockTime    int64          `json:"lock_time"`
}

func NewTimeLockMsg(from sdk.AccAddress, description string, amount sdk.Coins, lockTime int64) TimeLockMsg {
	return TimeLockMsg{
		From:        from,
		Description: description,
		Amount:      amount,
		LockTime:    lockTime,
	}
}

func (msg TimeLockMsg) Route() string { return MsgRoute }
func (msg TimeLockMsg) Type() string  { return "timeLock" }
func (msg TimeLockMsg) String() string {
	return fmt.Sprintf("TimeLock{%v#%v#%v#%v}", msg.From, msg.Description, msg.Amount, msg.LockTime)
}
func (msg TimeLockMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg TimeLockMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg TimeLockMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.From.String())
	}

	if len(msg.Description) <= 0 || len(msg.Description) > MaxDescriptionLength {
		return ErrInvalidDescription(DefaultCodespace,
			fmt.Sprintf("length of description(%d) should be larger than 0 and be less than or equal to %d",
				len(msg.Description), MaxDescriptionLength))
	}

	if msg.LockTime <= 0 {
		return ErrInvalidLockTime(DefaultCodespace, fmt.Sprintf("lock time(%d) should be larger than 0", msg.LockTime))
	}

	if !msg.Amount.IsValid() {
		return sdk.ErrInvalidCoins(msg.Amount.String())
	}

	if !msg.Amount.IsPositive() {
		return sdk.ErrInvalidCoins(msg.Amount.String())
	}

	return nil
}

func (msg TimeLockMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = TimeRelockMsg{}

type TimeRelockMsg struct {
	From        sdk.AccAddress `json:"from"`
	Id          int64          `json:"id"`
	Description string         `json:"description"`
	Amount      sdk.Coins      `json:"amount"`
	LockTime    int64          `json:"lock_time"`
}

func NewTimeRelockMsg(from sdk.AccAddress, id int64, description string, amount sdk.Coins, lockTime int64) TimeRelockMsg {
	return TimeRelockMsg{
		From:        from,
		Id:          id,
		Description: description,
		Amount:      amount,
		LockTime:    lockTime,
	}
}

func (msg TimeRelockMsg) Route() string { return MsgRoute }
func (msg TimeRelockMsg) Type() string  { return "timeRelock" }
func (msg TimeRelockMsg) String() string {
	return fmt.Sprintf("TimeRelock{%v#%v#%v#%v#%v}", msg.Id, msg.From, msg.Description, msg.Amount, msg.LockTime)
}
func (msg TimeRelockMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg TimeRelockMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg TimeRelockMsg) ValidateBasic() sdk.Error {
	if msg.Id < InitialRecordId {
		return ErrInvalidTimeLockId(DefaultCodespace, fmt.Sprintf("time lock id should not be less than %d", InitialRecordId))
	}

	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.From.String())
	}

	if len(msg.Description) > MaxDescriptionLength {
		return ErrInvalidDescription(DefaultCodespace,
			fmt.Sprintf("length of description(%d) should be less than or equal to %d",
				len(msg.Description), MaxDescriptionLength))
	}

	if msg.LockTime < 0 {
		return ErrInvalidLockTime(DefaultCodespace, fmt.Sprintf("lock time(%d) should not be less than 0", msg.LockTime))
	}

	if !msg.Amount.IsValid() {
		return sdk.ErrInvalidCoins(msg.Amount.String())
	}

	if !msg.Amount.IsNotNegative() {
		return sdk.ErrInvalidCoins(msg.Amount.String())
	}

	if len(msg.Description) == 0 &&
		msg.Amount.IsZero() &&
		msg.LockTime == 0 {
		return ErrInvalidRelock(DefaultCodespace, fmt.Sprintf("nothing to update for time lock"))
	}

	return nil
}

func (msg TimeRelockMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

type TimeUnlockMsg struct {
	From sdk.AccAddress `json:"from"`
	Id   int64          `json:"id"`
}

func NewTimeUnlockMsg(from sdk.AccAddress, id int64) TimeUnlockMsg {
	return TimeUnlockMsg{
		From: from,
		Id:   id,
	}
}

func (msg TimeUnlockMsg) Route() string { return MsgRoute }
func (msg TimeUnlockMsg) Type() string  { return "timeUnlock" }
func (msg TimeUnlockMsg) String() string {
	return fmt.Sprintf("TimeUnlock{%v#%v}", msg.From, msg.Id)
}
func (msg TimeUnlockMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg TimeUnlockMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg TimeUnlockMsg) ValidateBasic() sdk.Error {
	if msg.Id < InitialRecordId {
		return ErrInvalidTimeLockId(DefaultCodespace, fmt.Sprintf("time lock id should not be less than %d", InitialRecordId))
	}

	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.From.String())
	}
	return nil
}

func (msg TimeUnlockMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
