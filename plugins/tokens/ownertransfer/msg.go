package ownertransfer

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

const (
	Route                    = "tokensOwnershipTransfer"
	TransferOwnershipMsgType = "transferOwnership"
)

var _ sdk.Msg = TransferOwnershipMsg{}

type TransferOwnershipMsg struct {
	From     sdk.AccAddress `json:"from"`
	Symbol   string         `json:"symbol"`
	NewOwner sdk.AccAddress `json:"new_owner"`
}

func NewTransferOwnershipMsg(from sdk.AccAddress, symbol string, newOwner sdk.AccAddress) TransferOwnershipMsg {
	return TransferOwnershipMsg{
		From:     from,
		Symbol:   symbol,
		NewOwner: newOwner,
	}
}

func (msg TransferOwnershipMsg) Route() string  { return Route }
func (msg TransferOwnershipMsg) Type() string   { return TransferOwnershipMsgType }
func (msg TransferOwnershipMsg) String() string { return fmt.Sprintf("TransferOwnershipMsg{%#v}", msg) }

func (msg TransferOwnershipMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Invalid from address, expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.NewOwner) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Invalid newOwner, expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.NewOwner)))
	}

	if !types.IsValidMiniTokenSymbol(msg.Symbol) {
		err := types.ValidateTokenSymbol(msg.Symbol)
		if err != nil {
			return sdk.ErrInvalidCoins(err.Error())
		}
	}
	return nil
}

func (msg TransferOwnershipMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg TransferOwnershipMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.From}
}

func (msg TransferOwnershipMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
