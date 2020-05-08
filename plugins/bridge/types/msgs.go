package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	RouteBridge = "bridge"

	BindMsgType        = "crossBind"
	TransferOutMsgType = "crossTransferOut"
)

type RefundReason uint16

const (
	UnboundToken        RefundReason = 1
	Timeout             RefundReason = 2
	InsufficientBalance RefundReason = 3
	Unknown             RefundReason = 4
)

func (reason RefundReason) String() string {
	switch reason {
	case UnboundToken:
		return "UnboundToken"
	case Timeout:
		return "Timeout"
	case InsufficientBalance:
		return "InsufficientBalance"
	case Unknown:
		return "Unknown"
	default:
		return ""
	}
}

func ParseRefundReason(input string) (RefundReason, error) {
	switch strings.ToLower(input) {
	case "unboundtoken":
		return UnboundToken, nil
	case "timeout":
		return Timeout, nil
	case "insufficientbalance":
		return InsufficientBalance, nil
	case "unknown":
		return Unknown, nil
	default:
		return RefundReason(0), fmt.Errorf("unrecognized refund reason")
	}
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
		return ErrInvalidDecimal(fmt.Sprintf("decimals should be no less than 0"))
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

type BindStatus int8

const (
	BindStatusSuccess          BindStatus = 0
	BindStatusRejected         BindStatus = 1
	BindStatusTimeout          BindStatus = 2
	BindStatusInvalidParameter BindStatus = 3
)

func (status BindStatus) String() string {
	switch status {
	case BindStatusSuccess:
		return "UnboundToken"
	case BindStatusRejected:
		return "Timeout"
	case BindStatusTimeout:
		return "InsufficientBalance"
	case BindStatusInvalidParameter:
		return "InsufficientBalance"
	default:
		return ""
	}
}

func ParseBindStatus(input string) (BindStatus, error) {
	switch strings.ToLower(input) {
	case "success":
		return BindStatusSuccess, nil
	case "rejected":
		return BindStatusRejected, nil
	case "timeout":
		return BindStatusTimeout, nil
	case "invalidparameter":
		return BindStatusInvalidParameter, nil
	default:
		return BindStatus(-1), fmt.Errorf("unrecognized bind status")
	}
}

var _ sdk.Msg = TransferOutMsg{}

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
	return fmt.Sprintf("TransferOut{%v#%s#%s#%d}", msg.From, msg.To.String(), msg.Amount.String(), msg.ExpireTime)
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
		uint8   erc20Decimals;   // 1  116:117
		uint64  expireTime;      // 8  117:125
		uint256 relayReward;     // 32 125:157
	}
*/
func SerializeBindPackage(bep2TokenSymbol string, contractAddr []byte,
	totalSupply sdk.Int, peggyAmount sdk.Int, decimals int8, expireTime int64, relayReward sdk.Int) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+32+32+1+8+32)
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

	copy(serializedBytes[116:117], []byte{byte(decimals)})

	binary.BigEndian.PutUint64(serializedBytes[117:125], uint64(expireTime))

	length = len(relayReward.BigInt().Bytes())
	copy(serializedBytes[157-length:157], relayReward.BigInt().Bytes())

	return serializedBytes, nil
}

/*
	struct RefundPackage {
        uint256 refundAmount;       // 32 0:32
        address contractAddr;       // 20 32:52
        address payable refundAddr; // 20 52:72
		uint16 refundReason         // 2  72:74
    }
*/
func SerializeTransferInFailurePackage(refundAmount sdk.Int, contractAddr []byte, refundAddr []byte, refundReason RefundReason) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20+2)
	if len(contractAddr) != 20 || len(refundAddr) != 20 {
		return nil, fmt.Errorf("length of address must be 20")
	}
	if refundAmount.BigInt().BitLen() > 255 {
		return nil, fmt.Errorf("amount overflow, maximum bits is 256")
	}
	length := len(refundAmount.BigInt().Bytes())
	copy(serializedBytes[32-length:32], refundAmount.BigInt().Bytes())

	copy(serializedBytes[32:52], contractAddr)
	copy(serializedBytes[52:72], refundAddr)
	binary.BigEndian.PutUint16(serializedBytes[72:74], uint16(refundReason))

	return serializedBytes, nil
}

/*
	struct CrossChainTransferPackage {
        bytes32 bep2TokenSymbol;    // 32 0:32
        address contractAddr;       // 20 32:52
        address refundAddr;         // 20 52:72
        address payable recipient;  // 20 72:92
        uint256 amount;             // 32 92:124
        uint64  expireTime;         // 8  124:132
        uint256 relayReward;        // 32 132:164
    }
*/
func SerializeTransferOutPackage(bep2TokenSymbol string, contractAddr []byte, refundAddr []byte, recipient []byte,
	amount sdk.Int, expireTime int64, relayReward sdk.Int) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20+20+32+8+32)
	if len(bep2TokenSymbol) > 32 {
		return nil, fmt.Errorf("bep2 token symbol length should be no more than 32")
	}
	copy(serializedBytes[0:32], bep2TokenSymbol)

	if len(contractAddr) != 20 || len(refundAddr) != 20 || len(recipient) != 20 {
		return nil, fmt.Errorf("length of address must be 20")
	}
	copy(serializedBytes[32:52], contractAddr)
	copy(serializedBytes[52:72], refundAddr)
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
