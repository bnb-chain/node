package issue

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route  = "tokens/issue"
const (
	IssueTinyMsgType = "tinyIssueMsg"
)

var _ sdk.Msg = IssueTinyMsg{}

type IssueTinyMsg struct {
	IssueMiniMsg
}

func NewIssueTinyMsg(from sdk.AccAddress, name, symbol string, supply int64, mintable bool, tokenURI string) IssueTinyMsg {
	return IssueTinyMsg{IssueMiniMsg{
		From:        from,
		Name:        name,
		Symbol:      symbol,
		TotalSupply: supply,
		Mintable:    mintable,
		TokenURI:    tokenURI,
	},
	}
}

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg IssueTinyMsg) ValidateBasic() sdk.Error {
	if msg.From == nil {
		return sdk.ErrInvalidAddress("sender address cannot be empty")
	}

	if err := types.ValidateIssueMsgMiniTokenSymbol(msg.Symbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	if len(msg.Name) == 0 || len(msg.Name) > maxTokenNameLength {
		return sdk.ErrInvalidCoins(fmt.Sprintf("token name should have 1 ~ %v characters", maxTokenNameLength))
	}

	if len(msg.TokenURI) > types.MaxTokenURILength {
		return sdk.ErrInvalidCoins(fmt.Sprintf("token seturi should not exceed %v characters", types.MaxTokenURILength))
	}

	if msg.TotalSupply < types.MiniTokenMinExecutionAmount || msg.TotalSupply > types.TinyRangeType.UpperBound() {
		return sdk.ErrInvalidCoins(fmt.Sprintf("total supply should be between %d and %d", types.MiniTokenMinExecutionAmount, types.TinyRangeType.UpperBound()))
	}

	return nil
}

// Implements IssueTinyMsg.
func (msg IssueTinyMsg) Route() string { return Route }
func (msg IssueTinyMsg) Type() string {
	return IssueTinyMsgType
}

func (msg IssueTinyMsg) String() string               { return fmt.Sprintf("IssueTinyMsg{%#v}", msg) }
func (msg IssueTinyMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }
func (msg IssueTinyMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
func (msg IssueTinyMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
