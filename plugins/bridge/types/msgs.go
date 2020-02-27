package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	RouteTransfer = "crossTransfer"
)

var _ sdk.Msg = TransferMsg{}

type TransferMsg struct {
	Symbol           string          `json:"symbol"`
	Sequence         int64           `json:"nonce"`
	ContractAddress  EthereumAddress `json:"contract_address"`
	SenderAddress    EthereumAddress `json:"sender_address"`
	ReceiverAddress  sdk.AccAddress  `json:"receiver_address"`
	Amount           sdk.Coin        `json:"amount"`
	RelayFee         sdk.Coin        `json:"relay_fee"`
	ValidatorAddress sdk.ValAddress  `json:"validator_address"`
}

func NewTransferMsg(symbol string, sequence int64, contractAddr EthereumAddress,
	senderAddr EthereumAddress, receiverAddr sdk.AccAddress, amount sdk.Coin,
	relayFee sdk.Coin, validatorAddr sdk.ValAddress) TransferMsg {
	return TransferMsg{
		Symbol:           symbol,
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
func (msg TransferMsg) Route() string { return RouteTransfer }
func (msg TransferMsg) Type() string  { return RouteTransfer }
func (msg TransferMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.ValidatorAddress)}
}

func (msg TransferMsg) String() string {
	return fmt.Sprintf("TransferMsg{Symbol:%s,"+
		"ValidatorAddress:%v,"+
		"ContractAddress:%s,"+
		"SenderAddress:%s,"+
		"ReceiverAddress:%s,"+
		"Amount:%s,"+
		"RelayFee:%s,"+
		"ValidatorAddress:%s}", msg.Symbol, msg.ValidatorAddress,
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
	if len(msg.Symbol) == 0 {
		return ErrInvalidSymbol("symbol should not be empty")
	}
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
