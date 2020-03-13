package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxDecimal                  int8 = 18
	MinTransferOutExpireTimeGap      = 60 * time.Second
	MinBindExpireTimeGap             = 600 * time.Second
	// TODO change relay reward, relay reward should have 18 decimals
	RelayReward int64 = 1e6

	BindChannelName        = "bind"
	TransferOutChannelName = "transferOut"
	TimeoutChannelName     = "timeout"

	RouteBridge = "bridge"

	TransferInMsgType         = "crossTransferIn"
	TransferOutTimeoutMsgType = "crossTransferOutTimeout"
	BindMsgType               = "crossBind"
	TransferOutMsgType        = "crossTransferOut"
	UpdateBindMsgType         = "crossUpdateBind"
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

func NewTransferInMsg(sequence int64, contractAddr EthereumAddress,
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
	return fmt.Sprintf("TransferIn{%v#%s#%s#%s#%s#%s#%s#%d}",
		msg.ValidatorAddress, msg.ContractAddress.String(), msg.SenderAddress.String(), msg.ReceiverAddress.String(),
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
		return ErrInvalidAmount("relay fee amount should be positive")
	}
	return nil
}

var _ sdk.Msg = TransferOutTimeoutMsg{}

type TransferOutTimeoutMsg struct {
	SenderAddress    sdk.AccAddress `json:"sender_address"`
	Sequence         int64          `json:"sequence"`
	Amount           sdk.Coin       `json:"amount"`
	ValidatorAddress sdk.AccAddress `json:"validator_address"`
}

func NewTimeoutMsg(senderAddr sdk.AccAddress, sequence int64, amount sdk.Coin, validatorAddr sdk.AccAddress) TransferOutTimeoutMsg {
	return TransferOutTimeoutMsg{
		SenderAddress:    senderAddr,
		Sequence:         sequence,
		Amount:           amount,
		ValidatorAddress: validatorAddr,
	}
}

// nolint
func (msg TransferOutTimeoutMsg) Route() string { return RouteBridge }
func (msg TransferOutTimeoutMsg) Type() string  { return TransferOutTimeoutMsgType }
func (msg TransferOutTimeoutMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.ValidatorAddress}
}
func (msg TransferOutTimeoutMsg) String() string {
	return fmt.Sprintf("TransferOutTimeout{%s#%d#%s#%s}",
		msg.SenderAddress.String(), msg.Sequence, msg.Amount.String(), msg.ValidatorAddress.String())
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg TransferOutTimeoutMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg TransferOutTimeoutMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg TransferOutTimeoutMsg) ValidateBasic() sdk.Error {
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
	From             sdk.AccAddress  `json:"from"`
	Symbol           string          `json:"symbol"`
	Amount           int64           `json:"amount"`
	ContractAddress  EthereumAddress `json:"contract_address"`
	ContractDecimals int8            `json:"contract_decimals"`
	ExpireTime       int64           `json:"expire_time"`
}

func NewBindMsg(from sdk.AccAddress, symbol string, amount int64, contractAddress EthereumAddress, contractDecimals int8, expireTime int64) BindMsg {
	return BindMsg{
		From:             from,
		Amount:           amount,
		Symbol:           symbol,
		ContractAddress:  contractAddress,
		ContractDecimals: contractDecimals,
		ExpireTime:       expireTime,
	}
}

func (msg BindMsg) Route() string { return RouteBridge }
func (msg BindMsg) Type() string  { return BindMsgType }
func (msg BindMsg) String() string {
	return fmt.Sprintf("Bind{%v#%s#%d$%s#%d#%d}", msg.From, msg.Symbol, msg.Amount, msg.ContractAddress.String(), msg.ContractDecimals, msg.ExpireTime)
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

	if msg.ContractDecimals < 0 {
		return ErrInvalidDecimal("decimal should be no less than 0")
	}

	if msg.ExpireTime <= 0 {
		return ErrInvalidExpireTime("expire time should be larger than 0")
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

var _ sdk.Msg = UpdateBindMsg{}

type BindStatus int8

const (
	BindStatusSuccess          BindStatus = 0
	BindStatusRejected         BindStatus = 1
	BindStatusTimeout          BindStatus = 2
	BindStatusInvalidParameter BindStatus = 3
)

func ParseBindStatus(input string) (BindStatus, error) {
	switch strings.ToLower(input) {
	case "success":
		return BindStatusSuccess, nil
	case "reject":
		return BindStatusRejected, nil
	case "timeout":
		return BindStatusTimeout, nil
	case "invalid" :
		return BindStatusInvalidParameter, nil
	default:
		return BindStatusInvalidParameter, fmt.Errorf("unsupported bind status")
	}
}

type UpdateBindMsg struct {
	Sequence         int64           `json:"sequence"`
	Status           BindStatus      `json:"status"`
	Symbol           string          `json:"symbol"`
	Amount           int64           `json:"amount"`
	ContractAddress  EthereumAddress `json:"contract_address"`
	ContractDecimals int8            `json:"contract_decimals"`
	ValidatorAddress sdk.AccAddress  `json:"validator_address"`
}

func NewUpdateBindMsg(sequence int64, validatorAddress sdk.AccAddress, symbol string, amount int64, contractAddress EthereumAddress, contractDecimals int8, status BindStatus) UpdateBindMsg {
	return UpdateBindMsg{
		Sequence:         sequence,
		ValidatorAddress: validatorAddress,
		Amount:           amount,
		Symbol:           symbol,
		ContractAddress:  contractAddress,
		ContractDecimals: contractDecimals,
		Status:           status,
	}
}

func (msg UpdateBindMsg) Route() string { return RouteBridge }
func (msg UpdateBindMsg) Type() string  { return UpdateBindMsgType }
func (msg UpdateBindMsg) String() string {
	return fmt.Sprintf("UpdateBind{%v#%s#%d$%s#%d#%d}", msg.ValidatorAddress, msg.Symbol, msg.Amount, msg.ContractAddress.String(), msg.ContractDecimals, msg.Status)
}
func (msg UpdateBindMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg UpdateBindMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.ValidatorAddress} }

func (msg UpdateBindMsg) ValidateBasic() sdk.Error {
	if len(msg.ValidatorAddress) != sdk.AddrLen {
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

	if msg.ContractDecimals < 0 {
		return ErrInvalidDecimal("decimal should be no less than 0")
	}

	return nil
}
func (msg UpdateBindMsg) GetSignBytes() []byte {
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
		From:       from,
		To:         to,
		Amount:     amount,
		ExpireTime: expireTime,
	}
}

func (msg TransferOutMsg) Route() string { return RouteBridge }
func (msg TransferOutMsg) Type() string  { return TransferOutMsgType }
func (msg TransferOutMsg) String() string {
	return fmt.Sprintf("Transfer{%v#%s#%s#%d}", msg.From, msg.To.String(), msg.Amount.String(), msg.ExpireTime)
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

/*
	struct BindRequestPackage {
		bytes32 bep2TokenSymbol; // 32 0:32
		address contractAddr;    // 20 32:52
		uint256 totalSupply;     // 32 52:84
		uint256 peggyAmount;     // 32 84:116
		uint64  expireTime;      // 8  116:124
		uint256 relayReward;     // 32 124:156
	}
*/
func SerializeBindPackage(bep2TokenSymbol string, contractAddr []byte,
	totalSupply sdk.Int, peggyAmount sdk.Int, expireTime int64, relayReward sdk.Int) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+32+32+8+32)
	if len(bep2TokenSymbol) > 32 {
		return nil, fmt.Errorf("bep2 token symbol length should be no more than 32")
	}
	copy(serializedBytes[0:32], bep2TokenSymbol)

	if len(contractAddr) != 20 {
		return nil, fmt.Errorf("contract address length must be 20")
	}
	copy(serializedBytes[32:52], contractAddr)
	if totalSupply.BigInt().BitLen() > 255 || peggyAmount.BigInt().BitLen() > 255 || relayReward.BigInt().BitLen() > 255 {
		return nil, fmt.Errorf("overflow, maximum bits is 255")
	}
	length := len(totalSupply.BigInt().Bytes())
	copy(serializedBytes[84-length:84], totalSupply.BigInt().Bytes())

	length = len(peggyAmount.BigInt().Bytes())
	copy(serializedBytes[116-length:116], peggyAmount.BigInt().Bytes())

	binary.BigEndian.PutUint64(serializedBytes[116:124], uint64(expireTime))

	length = len(relayReward.BigInt().Bytes())
	copy(serializedBytes[156-length:156], relayReward.BigInt().Bytes())

	return serializedBytes, nil
}

/*
	struct TimeoutPackage {
        uint256 refundAmount;       // 32 0:32
        address contractAddr;       // 20 32:52
        address payable refundAddr; // 20 52:72
    }
*/
func SerializeTimeoutPackage(refundAmount sdk.Int, contractAddr []byte, refundAddr []byte) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20)
	if len(contractAddr) != 20 || len(refundAddr) != 20 {
		return nil, fmt.Errorf("length of address must be 20")
	}
	if refundAmount.BigInt().BitLen() > 255 {
		return nil, fmt.Errorf("amount overflow, maximum bits is 256")
	}
	length := len(refundAmount.BigInt().Bytes())
	copy(serializedBytes[32-length:32], refundAmount.BigInt().Bytes())

	copy(serializedBytes[32:52], contractAddr)
	copy(serializedBytes[52:], refundAddr)

	return serializedBytes, nil
}

/*
	struct CrossChainTransferPackage {
        bytes32 bep2TokenSymbol;    // 32 0:32
        address contractAddr;       // 20 32:52
        address sender;             // 20 52:72
        address payable recipient;  // 20 72:92
        uint256 amount;             // 32 92:124
        uint64  expireTime;         // 8  124:132
        uint256 relayReward;        // 32 132:164
    }
*/
func SerializeTransferOutPackage(bep2TokenSymbol string, contractAddr []byte, sender []byte, recipient []byte,
	amount sdk.Int, expireTime int64, relayReward sdk.Int) ([]byte, error) {
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

	if amount.BigInt().BitLen() > 255 || relayReward.BigInt().BitLen() > 255 {
		return nil, fmt.Errorf("overflow, maximum bits is 255")
	}

	length := len(amount.BigInt().Bytes())
	copy(serializedBytes[124-length:124], amount.BigInt().Bytes())

	binary.BigEndian.PutUint64(serializedBytes[124:132], uint64(expireTime))

	length = len(relayReward.BigInt().Bytes())
	copy(serializedBytes[164-length:164], relayReward.BigInt().Bytes())

	return serializedBytes, nil
}
