package tx

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: merge this type into Tx
// Transactions messages must fulfill the Msg
type Msg interface {

	// Return the message type.
	// Must be alphanumeric or empty.
	Type() string

	// ValidateBasic does a simple validation check that
	// doesn't require access to any other information.
	ValidateBasic() sdk.Error

	// Get the canonical byte representation of the Msg.
	GetSignBytes() []byte

	// Signers returns the addrs of signers that must sign.
	// CONTRACT: All signatures must be present to be valid.
	// CONTRACT: Returns addrs in some deterministic order.
	GetSigners() []sdk.AccAddress
}

//__________________________________________________________

// Transactions objects must fulfill the Tx
type Tx interface {

	// Gets the Msg.
	GetMsg() Msg
}

//__________________________________________________________

// TxDecoder unmarshals transaction bytes
type TxDecoder func(txBytes []byte) (Tx, sdk.Error)

//__________________________________________________________

var _ Msg = (*TestMsg)(nil)

// msg type for testing
type TestMsg struct {
	signers []sdk.AccAddress
}

func NewTestMsg(addrs ...sdk.AccAddress) *TestMsg {
	return &TestMsg{
		signers: addrs,
	}
}

//nolint
func (msg *TestMsg) Type() string { return "TestMsg" }
func (msg *TestMsg) GetSignBytes() []byte {
	bz, err := json.Marshal(msg.signers)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}
func (msg *TestMsg) ValidateBasic() sdk.Error { return nil }
func (msg *TestMsg) GetSigners() []sdk.AccAddress {
	return msg.signers
}
