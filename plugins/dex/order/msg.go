package order

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const Route = "dexOrder"

// A really cool msg type, these fields are can be entirely arbitrary and
// custom to your message
type SetTrendMsg struct {
	Sender sdk.Address
	Cool   string
}

type MakeOfferMsg struct {
	Sender sdk.Address
}

type FillOfferMsg struct {
	Sender sdk.Address
}

type CancelOfferMsg struct {
	Sender sdk.Address
}

// NewMakeOfferMsg - Creates a new MakeOfferMsg
func NewMakeOfferMsg(sender sdk.Address) MakeOfferMsg {
	return MakeOfferMsg{
		Sender: sender,
	}
}

var _ sdk.Msg = MakeOfferMsg{}

// nolint
func (msg MakeOfferMsg) Type() string                            { return "dex" }
func (msg MakeOfferMsg) Get(key interface{}) (value interface{}) { return nil }
func (msg MakeOfferMsg) GetSigners() []sdk.Address               { return []sdk.Address{msg.Sender} }
func (msg MakeOfferMsg) String() string {
	return fmt.Sprintf("MakeOfferMsg{Sender: %v}", msg.Sender)
}

// NewFillOfferMsg - Creates a new FillOfferMsg
func NewFillOfferMsg(sender sdk.Address) FillOfferMsg {
	return FillOfferMsg{
		Sender: sender,
	}
}

var _ sdk.Msg = FillOfferMsg{}

// nolint
func (msg FillOfferMsg) Type() string                            { return "dex" }
func (msg FillOfferMsg) Get(key interface{}) (value interface{}) { return nil }
func (msg FillOfferMsg) GetSigners() []sdk.Address               { return []sdk.Address{msg.Sender} }
func (msg FillOfferMsg) String() string {
	return fmt.Sprintf("FillOfferMsg{Sender: %v}", msg.Sender)
}

// NewCancelOfferMsg - Creates a new CancelOfferMsg
func NewCancelOfferMsg(sender sdk.Address) CancelOfferMsg {
	return CancelOfferMsg{
		Sender: sender,
	}
}

var _ sdk.Msg = CancelOfferMsg{}

// nolint
func (msg CancelOfferMsg) Type() string                            { return "dex" }
func (msg CancelOfferMsg) Get(key interface{}) (value interface{}) { return nil }
func (msg CancelOfferMsg) GetSigners() []sdk.Address               { return []sdk.Address{msg.Sender} }
func (msg CancelOfferMsg) String() string {
	return fmt.Sprintf("CancelOfferMsg{Sender: %v}", msg.Sender)
}

// TODO: validate messages

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg MakeOfferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg FillOfferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg CancelOfferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg MakeOfferMsg) ValidateBasic() sdk.Error {
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	return nil
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg FillOfferMsg) ValidateBasic() sdk.Error {
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	return nil
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg CancelOfferMsg) ValidateBasic() sdk.Error {
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	return nil
}
