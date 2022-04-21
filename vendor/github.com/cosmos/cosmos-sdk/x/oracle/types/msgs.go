package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

const (
	RouteOracle = "oracle"

	ClaimMsgType = "oracleClaim"
)

var _ sdk.Msg = ClaimMsg{}

type Packages []Package

type Package struct {
	ChannelId sdk.ChannelID
	Sequence  uint64
	Payload   []byte
}

type ClaimMsg struct {
	ChainId          sdk.ChainID    `json:"chain_id"`
	Sequence         uint64         `json:"sequence"`
	Payload          []byte         `json:"payload"`
	ValidatorAddress sdk.AccAddress `json:"validator_address"`
}

func NewClaimMsg(ChainId sdk.ChainID, sequence uint64, payload []byte, validatorAddr sdk.AccAddress) ClaimMsg {
	return ClaimMsg{
		ChainId:          ChainId,
		Sequence:         sequence,
		Payload:          payload,
		ValidatorAddress: validatorAddr,
	}
}

// nolint
func (msg ClaimMsg) Route() string { return RouteOracle }
func (msg ClaimMsg) Type() string  { return ClaimMsgType }
func (msg ClaimMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.ValidatorAddress}
}

func (msg ClaimMsg) String() string {
	return fmt.Sprintf("Claim{%v#%v#%v#%x}",
		msg.ChainId, msg.Sequence, msg.ValidatorAddress.String(), msg.Payload)
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg ClaimMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg ClaimMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg ClaimMsg) ValidateBasic() sdk.Error {
	if len(msg.Payload) < types.PackageHeaderLength {
		return ErrInvalidPayloadHeader(fmt.Sprintf("length of payload is less than %d", types.PackageHeaderLength))
	}
	if len(msg.ValidatorAddress) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.ValidatorAddress.String())
	}
	return nil
}
