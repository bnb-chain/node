package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxDecimal                  int = 18
	MinTransferOutExpireTimeGap     = 60 * time.Second
	// TODO change relay reward
	RelayReward int64 = 1e6

	BindChannelName        = "bind"
	TransferOutChannelName = "transferOut"
	TimeoutChannelName     = "timeout"

	RouteBridge = "bridge"

	TransferInMsgType  = "crossTransferIn"
	TimeoutMsgType     = "crossTimeout"
	BindMsgType        = "crossBind"
	TransferOutMsgType = "crossTransferOut"
)

var _ sdk.Msg = TransferInMsg{}

type TransferInMsg struct {
	Sequence         int64           `json:"sequence"`
	ContractAddress  EthereumAddress `json:"contract_address"`
	SenderAddress    EthereumAddress `json:"sender_address"`
	ReceiverAddress  sdk.AccAddress  `json:"receiver_address"`
	Amount           sdk.Coin        `json:"amount"`
	RelayFee         sdk.Coin        `json:"relay_fee"`
	ValidatorAddress sdk.AccAddress  `json:"validator_address"`
	ExpireTime       int64           `json:"expire_time"`
}

func NewTransferMsg(sequence int64, contractAddr EthereumAddress,
	senderAddr EthereumAddress, receiverAddr sdk.AccAddress, amount sdk.Coin,
	relayFee sdk.Coin, validatorAddr sdk.AccAddress, expireTime int64) TransferInMsg {
	return TransferInMsg{
		Sequence:         sequence,
		ContractAddress:  contractAddr,
		SenderAddress:    senderAddr,
		ReceiverAddress:  receiverAddr,
		Amount:           amount,
		RelayFee:         relayFee,
		ValidatorAddress: validatorAddr,
		ExpireTime:       expireTime,
	}
}

// nolint
func (msg TransferInMsg) Route() string { return RouteBridge }
func (msg TransferInMsg) Type() string  { return TransferInMsgType }
func (msg TransferInMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.ValidatorAddress}
}

func (msg TransferInMsg) String() string {
	return fmt.Sprintf("TransferInMsg{"+
		"ValidatorAddress:%v,"+
		"ContractAddress:%s,"+
		"SenderAddress:%s,"+
		"ReceiverAddress:%s,"+
		"Amount:%s,"+
		"RelayFee:%s,"+
		"ValidatorAddress:%s,"+
		"ExpireTime:%d}", msg.ValidatorAddress,
		msg.ContractAddress.String(), msg.SenderAddress.String(), msg.ReceiverAddress.String(),
		msg.Amount.String(), msg.RelayFee.String(), msg.ValidatorAddress.String(), msg.ExpireTime)
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg TransferInMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg TransferInMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg TransferInMsg) ValidateBasic() sdk.Error {
	if msg.Sequence < 0 {
		return ErrInvalidSequence("sequence should not be less than 0")
	}
	if msg.ExpireTime <= 0 {
		return ErrInvalidExpireTime("expire time should be larger than 0")
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
	return fmt.Sprintf("TransferInMsg{"+
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

var _ sdk.Msg = BindMsg{}

type BindMsg struct {
	From            sdk.AccAddress  `json:"from"`
	Symbol          string          `json:"symbol"`
	Amount          int64           `json:"amount"`
	ContractAddress EthereumAddress `json:"contract_address"`
	ContractDecimal int             `json:"contract_decimal"`
}

func NewBindMsg(from sdk.AccAddress, symbol string, amount int64, contractAddress EthereumAddress, contractDecimal int) BindMsg {
	return BindMsg{
		From:            from,
		Amount:          amount,
		Symbol:          symbol,
		ContractAddress: contractAddress,
		ContractDecimal: contractDecimal,
	}
}

func (msg BindMsg) Route() string { return RouteBridge }
func (msg BindMsg) Type() string  { return BindMsgType }
func (msg BindMsg) String() string {
	return fmt.Sprintf("Bind{%v#%s#%d$%s#%d}", msg.From, msg.Symbol, msg.Amount, msg.ContractAddress.String(), msg.ContractDecimal)
}
func (msg BindMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg BindMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg BindMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("address length should be %d", sdk.AddrLen))
	}

	if len(msg.Symbol) == 0 {
		return ErrInvalidSymbol("symbol should not be empty")
	}

	if msg.Amount <= 0 {
		return ErrInvalidAmount("amount should be larger than 0")
	}

	if msg.ContractAddress.IsEmpty() {
		return ErrInvalidContractAddress("contract address should not be empty")
	}

	if msg.ContractDecimal < 0 || msg.ContractDecimal > MaxDecimal {
		return ErrInvalidDecimal(fmt.Sprintf("decimal should be no less than 0 and larger than %d", MaxDecimal))
	}

	return nil
}

func (msg BindMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

type TransferOutMsg struct {
	From       sdk.AccAddress  `json:"from"`
	To         EthereumAddress `json:"to"`
	Amount     sdk.Coin        `json:"amount"`
	ExpireTime int64           `json:"expire_time"`
}

func NewTransferOutMsg(from sdk.AccAddress, to EthereumAddress, amount sdk.Coin, expireTime int64) TransferOutMsg {
	return TransferOutMsg{
		From:   from,
		To:     to,
		Amount: amount,
	}
}

func (msg TransferOutMsg) Route() string { return RouteBridge }
func (msg TransferOutMsg) Type() string  { return TransferOutMsgType }
func (msg TransferOutMsg) String() string {
	return fmt.Sprintf("Transfer{%v#%s#%s}", msg.From, msg.To.String(), msg.Amount.String())
}
func (msg TransferOutMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg TransferOutMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }
func (msg TransferOutMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("address length should be %d", sdk.AddrLen))
	}

	if msg.To.IsEmpty() {
		return ErrInvalidContractAddress("to address should not be empty")
	}

	if !msg.Amount.IsPositive() {
		return sdk.ErrInvalidCoins("amount should be positive")
	}

	if msg.ExpireTime <= 0 {
		return ErrInvalidExpireTime("expire time should be larger than 0")
	}

	return nil
}
func (msg TransferOutMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

func SerializeBindPackage(bep2TokenSymbol string, bep2TokenOwner sdk.AccAddress, contractAddr []byte,
	totalSupply int64, peggyAmount int64, relayReward int64) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20+32+32+8)
	if len(bep2TokenSymbol) > 32 {
		return nil, fmt.Errorf("bep2 token symbol length should be no more than 32")
	}
	copy(serializedBytes[0:32], bep2TokenSymbol)
	copy(serializedBytes[32:52], bep2TokenOwner)

	if len(contractAddr) != 20 {
		return nil, fmt.Errorf("contract address length must be 20")
	}
	copy(serializedBytes[52:72], contractAddr)

	binary.BigEndian.PutUint64(serializedBytes[96:104], uint64(totalSupply))
	binary.BigEndian.PutUint64(serializedBytes[128:136], uint64(peggyAmount))
	binary.BigEndian.PutUint64(serializedBytes[160:168], uint64(relayReward))

	return serializedBytes, nil
}

func SerializeTimeoutPackage(refundAmount int64, contractAddr []byte, refundAddr []byte) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20)
	if len(contractAddr) != 20 || len(refundAddr) != 20 {
		return nil, fmt.Errorf("length of address must be 20")
	}
	binary.BigEndian.PutUint64(serializedBytes[24:32], uint64(refundAmount))
	copy(serializedBytes[32:52], contractAddr)
	copy(serializedBytes[52:], refundAddr)

	return serializedBytes, nil
}

func SerializeTransferOutPackage(bep2TokenSymbol string, contractAddr []byte, sender []byte, recipient []byte,
	amount int64, expireTime int64, relayReward int64) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20+20+32+8+32)
	if len(bep2TokenSymbol) > 32 {
		return nil, fmt.Errorf("bep2 token symbol length should be no more than 32")
	}
	copy(serializedBytes[0:32], bep2TokenSymbol)

	if len(contractAddr) != 20 || len(sender) != 20 || len(recipient) != 20 {
		return nil, fmt.Errorf("length of address must be 20")
	}
	copy(serializedBytes[32:52], contractAddr)
	copy(serializedBytes[52:72], sender)
	copy(serializedBytes[72:92], recipient)

	binary.BigEndian.PutUint64(serializedBytes[116:124], uint64(amount))
	binary.BigEndian.PutUint64(serializedBytes[124:132], uint64(expireTime))
	binary.BigEndian.PutUint64(serializedBytes[156:164], uint64(relayReward))

	return serializedBytes, nil
}
