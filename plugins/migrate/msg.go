package migrate

import (
	"encoding/json"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	Route   = "recover"
	MsgType = "validator_ownership"
)

var _ sdk.Msg = ValidatorOwnerShip{}

func NewValidatorOwnerShipMsg(
	bscOperatorAddress common.Address,
) ValidatorOwnerShip {
	return ValidatorOwnerShip{
		BSCOperatorAddress: bscOperatorAddress,
	}
}

func newValidatorOwnerShipSignData(
	bscOperatorAddress common.Address) ValidatorOwnerShipSignData {
	return ValidatorOwnerShipSignData{
		BSCOperatorAddress: strings.ToLower(bscOperatorAddress.Hex()),
	}
}

type ValidatorOwnerShipSignData struct {
	BSCOperatorAddress string `json:"bsc_operator_address"`
}

type ValidatorOwnerShip struct {
	BSCOperatorAddress common.Address `json:"bsc_operator_address"`
}

// GetInvolvedAddresses implements types.Msg.
func (msg ValidatorOwnerShip) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// GetSignBytes implements types.Msg.
func (msg ValidatorOwnerShip) GetSignBytes() []byte {
	b, err := json.Marshal(newValidatorOwnerShipSignData(msg.BSCOperatorAddress))
	if err != nil {
		panic(err)
	}
	return b
}

// GetSigners implements types.Msg.
func (m ValidatorOwnerShip) GetSigners() []sdk.AccAddress {
	// This is not a real on-chain transaction
	// We can get signer from the public key.
	return []sdk.AccAddress{}
}

// Route implements types.Msg.
func (ValidatorOwnerShip) Route() string {
	return Route
}

// Type implements types.Msg.
func (ValidatorOwnerShip) Type() string {
	return MsgType
}

// ValidateBasic implements types.Msg.
func (msg ValidatorOwnerShip) ValidateBasic() sdk.Error {
	return nil
}
