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
	Route           = "miniTokensIssue"
	IssueMsgType    = "miniIssueMsg"
	AdvIssueMsgType = "advMiniIssueMsg" //For max total supply in range 2

	maxTokenNameLength = 32
)

var _ sdk.Msg = IssueMsg{}

type IssueMsg struct {
	From           sdk.AccAddress `json:"from"`
	Name           string         `json:"name"`
	Symbol         string         `json:"symbol"`
	MaxTotalSupply int64          `json:"max_total_supply"`
	TotalSupply    int64          `json:"total_supply"`
	Mintable       bool           `json:"mintable"`
	TokenURI       string         `json:"token_uri"`
}

func NewIssueMsg(from sdk.AccAddress, name, symbol string, maxTotalSupply, supply int64, mintable bool, tokenURI string) IssueMsg {
	return IssueMsg{
		From:           from,
		Name:           name,
		Symbol:         symbol,
		MaxTotalSupply: maxTotalSupply,
		TotalSupply:    supply,
		Mintable:       mintable,
		TokenURI:       tokenURI,
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

	if msg.MaxTotalSupply < types.MiniTokenMinTotalSupply || msg.MaxTotalSupply > types.MiniTokenMaxTotalSupplyUpperBound {
		return sdk.ErrInvalidCoins(fmt.Sprintf("max total supply should be between %d ~ %d", types.MiniTokenMinTotalSupply, types.MiniTokenMaxTotalSupplyUpperBound))
	}

	if msg.TotalSupply < types.MiniTokenMinTotalSupply || msg.TotalSupply > msg.MaxTotalSupply {
		return sdk.ErrInvalidCoins(fmt.Sprintf("total supply should be between %d ~ %d", types.MiniTokenMinTotalSupply, msg.MaxTotalSupply))
	}

	return nil
}

// Implements IssueMsg.
func (msg IssueMsg) Route() string { return Route }
func (msg IssueMsg) Type() string {
	if msg.MaxTotalSupply > types.MiniTokenSupplyRange1UpperBound {
		return AdvIssueMsgType
	} else {
		return IssueMsgType
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
