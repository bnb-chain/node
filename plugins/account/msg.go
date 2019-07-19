package account

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AccountFlagsRoute      = "accountFlags"
	SetAccountFlagsMsgType = "setAccountFlags"
)

var _ sdk.Msg = SetAccountFlagsMsg{}

type SetAccountFlagsMsg struct {
	From  sdk.AccAddress `json:"from"`
	Flags uint64         `json:"flags"`
}

func NewSetAccountFlagsMsg(from sdk.AccAddress, flags uint64) SetAccountFlagsMsg {
	return SetAccountFlagsMsg{
		From:  from,
		Flags: flags,
	}
}

func (msg SetAccountFlagsMsg) Route() string { return AccountFlagsRoute }
func (msg SetAccountFlagsMsg) Type() string  { return SetAccountFlagsMsgType }
func (msg SetAccountFlagsMsg) String() string {
	return fmt.Sprintf("setAccountFlags{%v#%v}", msg.From, msg.Flags)
}
func (msg SetAccountFlagsMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg SetAccountFlagsMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg SetAccountFlagsMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	return nil
}

func (msg SetAccountFlagsMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
