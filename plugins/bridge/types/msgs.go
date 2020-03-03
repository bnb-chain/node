package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	RouteBridge = "bridge"

	TransferMsgType = "crossTransfer"
	TimeoutMsgType  = "crossTimeout"
)

var _ sdk.Msg = TransferMsg{}

type TransferMsg struct {
	Sequence         int64           `json:"sequence"`
	ContractAddress  EthereumAddress `json:"contract_address"`
	SenderAddress    EthereumAddress `json:"sender_address"`
	ReceiverAddress  sdk.AccAddress  `json:"receiver_address"`
	Amount           sdk.Coin        `json:"amount"`
	RelayFee         sdk.Coin        `json:"relay_fee"`
	ValidatorAddress sdk.AccAddress  `json:"validator_address"`
}

func NewTransferMsg(sequence int64, contractAddr EthereumAddress,
	senderAddr EthereumAddress, receiverAddr sdk.AccAddress, amount sdk.Coin,
	relayFee sdk.Coin, validatorAddr sdk.AccAddress) TransferMsg {
	return TransferMsg{
		Sequence:         sequence,
		ContractAddress:  contractAddr,
		SenderAddress:    senderAddr,
		ReceiverAddress:  receiverAddr,
		Amount:           amount,
		RelayFee:         relayFee,
		ValidatorAddress: validatorAddr,
	}
}

// nolint
func (msg TransferMsg) Route() string { return RouteBridge }
func (msg TransferMsg) Type() string  { return TransferMsgType }
func (msg TransferMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.ValidatorAddress}
}

func (msg TransferMsg) String() string {
	return fmt.Sprintf("TransferMsg{"+
		"ValidatorAddress:%v,"+
		"ContractAddress:%s,"+
		"SenderAddress:%s,"+
		"ReceiverAddress:%s,"+
		"Amount:%s,"+
		"RelayFee:%s,"+
		"ValidatorAddress:%s}", msg.ValidatorAddress,
		msg.ContractAddress.String(), msg.SenderAddress.String(), msg.ReceiverAddress.String(),
		msg.Amount.String(), msg.RelayFee.String(), msg.ValidatorAddress.String())
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg TransferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg TransferMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg TransferMsg) ValidateBasic() sdk.Error {
	if msg.Sequence < 0 {
		return ErrInvalidSequence("sequence should not be less than 0")
	}
	if msg.ContractAddress.IsEmpty() {
		return ErrInvalidEthereumAddress("contract address should not be empty")
	}
	if msg.SenderAddress.IsEmpty() {
		return ErrInvalidEthereumAddress("sender address should not be empty")
	}
	if len(msg.ReceiverAddress) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.ReceiverAddress.String())
	}
	if len(msg.ValidatorAddress) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.ValidatorAddress.String())
	}
	if !msg.Amount.IsPositive() {
		return ErrInvalidAmount("amount to send should be positive")
	}
	if !msg.RelayFee.IsPositive() {
		return ErrInvalidAmount("amount to send should be positive")
	}
	return nil
}

var _ sdk.Msg = TimeoutMsg{}

type TimeoutMsg struct {
	SenderAddress    sdk.AccAddress `json:"sender_address"`
	Sequence         int64          `json:"sequence"`
	Amount           sdk.Coin       `json:"amount"`
	ValidatorAddress sdk.AccAddress `json:"validator_address"`
}

func NewTimeoutMsg(senderAddr sdk.AccAddress, sequence int64, amount sdk.Coin, validatorAddr sdk.AccAddress) TimeoutMsg {
	return TimeoutMsg{
		SenderAddress:    senderAddr,
		Sequence:         sequence,
		Amount:           amount,
		ValidatorAddress: validatorAddr,
	}
}

// nolint
func (msg TimeoutMsg) Route() string { return RouteBridge }
func (msg TimeoutMsg) Type() string  { return TimeoutMsgType }
func (msg TimeoutMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.ValidatorAddress}
}

func (msg TimeoutMsg) String() string {
	return fmt.Sprintf("TransferMsg{"+
		"SenderAddress:%s,"+
		"Sequence:%d,"+
		"Amount:%s,"+
		"ValidatorAddress:%s}",
		msg.SenderAddress.String(), msg.Sequence, msg.Amount.String(), msg.ValidatorAddress.String())
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg TimeoutMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg TimeoutMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg TimeoutMsg) ValidateBasic() sdk.Error {
	if len(msg.SenderAddress) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.SenderAddress.String())
	}
	if msg.Sequence < 0 {
		return ErrInvalidSequence("sequence should not be less than 0")
	}
	if len(msg.ValidatorAddress) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(msg.ValidatorAddress.String())
	}
	if !msg.Amount.IsPositive() {
		return ErrInvalidAmount("amount to send should be positive")
	}
	return nil
}
