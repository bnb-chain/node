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
	Route              = "miniTokensIssue"
	IssueTinyMsgType   = "tinyIssueMsg"
	IssueMiniMsgType   = "miniIssueMsg" //For max total supply in range 2
	maxTokenNameLength = 32
)

var _ sdk.Msg = IssueMsg{}

type IssueMsg struct {
	From        sdk.AccAddress `json:"from"`
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	TokenType   int8           `json:"token_type"`
	TotalSupply int64          `json:"total_supply"`
	Mintable    bool           `json:"mintable"`
	TokenURI    string         `json:"token_uri"`
}

func NewIssueMsg(from sdk.AccAddress, name, symbol string, tokenType int8, supply int64, mintable bool, tokenURI string) IssueMsg {
	return IssueMsg{
		From:        from,
		Name:        name,
		Symbol:      symbol,
		TokenType:   tokenType,
		TotalSupply: supply,
		Mintable:    mintable,
		TokenURI:    tokenURI,
	}
}

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg IssueMsg) ValidateBasic() sdk.Error {
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

	if msg.TokenType != int8(types.SupplyRange.MINI) || msg.TokenType != int8(types.SupplyRange.TINY) {
		return sdk.ErrInvalidCoins(fmt.Sprintf("token type should be %d or %d", int8(types.SupplyRange.MINI), int8(types.SupplyRange.TINY)))
	}

	if msg.TotalSupply < types.MiniTokenMinTotalSupply || msg.TotalSupply > types.SupplyRangeType(msg.TokenType).UpperBound() {
		return sdk.ErrInvalidCoins(fmt.Sprintf("total supply should be between %d ~ %d", types.MiniTokenMinTotalSupply,  types.SupplyRangeType(msg.TokenType).UpperBound()))
	}

	return nil
}

// Implements IssueMsg.
func (msg IssueMsg) Route() string { return Route }
func (msg IssueMsg) Type() string {
	switch types.SupplyRangeType(msg.TokenType) {
	case types.SupplyRange.TINY:
		return IssueTinyMsgType
	case types.SupplyRange.MINI:
		return IssueMiniMsgType
	default:
		return IssueMiniMsgType
	}
}
func (msg IssueMsg) String() string               { return fmt.Sprintf("IssueMsg{%#v}", msg) }
func (msg IssueMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }
func (msg IssueMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
func (msg IssueMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}