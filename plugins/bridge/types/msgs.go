package types

import (
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	RouteBridge = "bridge"

	BindMsgType        = "crossBind"
	UnbindMsgType      = "crossUnbind"
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
	From             sdk.AccAddress    `json:"from"`
	Symbol           string            `json:"symbol"`
	Amount           int64             `json:"amount"`
	ContractAddress  SmartChainAddress `json:"contract_address"`
	ContractDecimals int8              `json:"contract_decimals"`
	ExpireTime       int64             `json:"expire_time"`
}

func NewBindMsg(from sdk.AccAddress, symbol string, amount int64, contractAddress SmartChainAddress, contractDecimals int8, expireTime int64) BindMsg {
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

	if msg.Amount < 0 {
		return ErrInvalidAmount("amount should be no less than 0")
	}

	if msg.ContractAddress.IsEmpty() {
		return ErrInvalidContractAddress("contract address should not be empty")
	}

	if msg.ContractDecimals < 0 {
		return ErrInvalidDecimals(fmt.Sprintf("decimals should be no less than 0"))
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

var _ sdk.Msg = UnbindMsg{}

type UnbindMsg struct {
	From   sdk.AccAddress `json:"from"`
	Symbol string         `json:"symbol"`
}

func NewUnbindMsg(from sdk.AccAddress, symbol string) UnbindMsg {
	return UnbindMsg{
		From:   from,
		Symbol: symbol,
	}
}

func (msg UnbindMsg) Route() string { return RouteBridge }
func (msg UnbindMsg) Type() string  { return UnbindMsgType }
func (msg UnbindMsg) String() string {
	return fmt.Sprintf("Unbind{%v#%s}", msg.From, msg.Symbol)
}
func (msg UnbindMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg UnbindMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg UnbindMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("address length should be %d", sdk.AddrLen))
	}

	if len(msg.Symbol) == 0 {
		return ErrInvalidSymbol("symbol should not be empty")
	}

	return nil
}

func (msg UnbindMsg) GetSignBytes() []byte {
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
	From       sdk.AccAddress    `json:"from"`
	To         SmartChainAddress `json:"to"`
	Amount     sdk.Coin          `json:"amount"`
	ExpireTime int64             `json:"expire_time"`
}

func NewTransferOutMsg(from sdk.AccAddress, to SmartChainAddress, amount sdk.Coin, expireTime int64) TransferOutMsg {
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
