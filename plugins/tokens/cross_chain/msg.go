package cross_chain

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

const (
	MaxDecimal int = 18
)

const (
	CrossChainRoute = "crossChain"

	BindMsgType     = "bindToBSC"
	TransferMsgType = "transferToBSC"
)

var _ sdk.Msg = BindMsg{}

type BindMsg struct {
	From            sdk.AccAddress        `json:"from"`
	Symbol          string                `json:"symbol"`
	ContractAddress types.EthereumAddress `json:"contract_address"`
	ContractDecimal int                   `json:"contract_decimal"`
}

func NewBindMsg(from sdk.AccAddress, symbol string, contractAddress types.EthereumAddress, contractDecimal int) BindMsg {
	return BindMsg{
		From:            from,
		Symbol:          symbol,
		ContractAddress: contractAddress,
		ContractDecimal: contractDecimal,
	}
}

func (msg BindMsg) Route() string { return CrossChainRoute }
func (msg BindMsg) Type() string  { return BindMsgType }
func (msg BindMsg) String() string {
	return fmt.Sprintf("Bind{%v#%s%d}", msg.From, msg.ContractAddress.String(), msg.ContractDecimal)
}
func (msg BindMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg BindMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg BindMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("address length should be %d", sdk.AddrLen))
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

type TransferMsg struct {
	From            sdk.AccAddress        `json:"from"`
	ContractAddress types.EthereumAddress `json:"contract_address"`
	To              types.EthereumAddress `json:"to"`
	Amount          sdk.Coin              `json:"amount"`
}

func NewTransferMsg(from sdk.AccAddress, contractAddress types.EthereumAddress,
	to types.EthereumAddress, amount sdk.Coin) TransferMsg {

	return TransferMsg{
		From:            from,
		ContractAddress: contractAddress,
		To:              to,
		Amount:          amount,
	}
}

func (msg TransferMsg) Route() string { return CrossChainRoute }
func (msg TransferMsg) Type() string  { return TransferMsgType }
func (msg TransferMsg) String() string {
	return fmt.Sprintf("Transfer{%v#%s#%s}", msg.From, msg.ContractAddress.String(), msg.To.String())
}
func (msg TransferMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg TransferMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }
func (msg TransferMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("address length should be %d", sdk.AddrLen))
	}

	if msg.ContractAddress.IsEmpty() {
		return ErrInvalidContractAddress("contract address should not be empty")
	}

	if msg.To.IsEmpty() {
		return ErrInvalidContractAddress("to address should not be empty")
	}
	return nil
}
func (msg TransferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
