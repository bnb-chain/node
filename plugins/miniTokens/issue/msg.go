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
	MintMsgType     = "miniMintMsg"

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
		return sdk.ErrInvalidCoins(fmt.Sprintf("token uri should not exceed %v characters", types.MaxTokenURILength))
	}

	if msg.MaxTotalSupply%types.MiniTokenMinTotalSupply != 0 {
		return sdk.ErrInvalidCoins(fmt.Sprintf("max total supply should be a multiple of %v", types.MiniTokenMinTotalSupply))
	}

	if msg.TotalSupply%types.MiniTokenMinTotalSupply != 0 {
		return sdk.ErrInvalidCoins(fmt.Sprintf("total supply should be a multiple of %v", types.MiniTokenMinTotalSupply))
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

type MintMsg struct {
	From   sdk.AccAddress `json:"from"`
	Symbol string         `json:"symbol"`
	Amount int64          `json:"amount"`
}

func NewMintMsg(from sdk.AccAddress, symbol string, amount int64) MintMsg {
	return MintMsg{
		From:   from,
		Symbol: symbol,
		Amount: amount,
	}
}

func (msg MintMsg) ValidateBasic() sdk.Error {
	if msg.From == nil {
		return sdk.ErrInvalidAddress("sender address cannot be empty")
	}

	if err := types.ValidateMapperMiniTokenSymbol(msg.Symbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	if msg.Symbol == types.NativeTokenSymbol {
		return sdk.ErrInvalidCoins(fmt.Sprintf("cannot mint native token"))
	}

	if msg.Amount%types.MiniTokenMinTotalSupply != 0 {
		return sdk.ErrInvalidCoins(fmt.Sprintf("amount should be a multiple of %v", types.MiniTokenMinTotalSupply))
	}

	// handler will check:  msg.Amount + token.TotalSupply <= types.MaxTotalSupply
	if msg.Amount < types.MiniTokenMinTotalSupply || msg.Amount > types.MiniTokenMaxTotalSupplyUpperBound {
		return sdk.ErrInvalidCoins(fmt.Sprintf("Mint amount should be between %d ~ %d", types.MiniTokenMinTotalSupply, types.MiniTokenMaxTotalSupplyUpperBound))
	}

	return nil
}

// Implements MintMsg.
func (msg MintMsg) Route() string                { return Route }
func (msg MintMsg) Type() string                 { return MintMsgType }
func (msg MintMsg) String() string               { return fmt.Sprintf("MintMsg{%#v}", msg) }
func (msg MintMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }
func (msg MintMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
func (msg MintMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
