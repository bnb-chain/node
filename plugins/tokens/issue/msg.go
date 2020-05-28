package issue

import (
	"encoding/json"
	"fmt"

	"github.com/binance-chain/node/common/upgrade"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route  = "tokens/issue"
const (
	Route        = "tokensIssue"
	IssueMsgType = "issueMsg"
	MintMsgType  = "mintMsg"

	maxTokenNameLength = 32
)

var _ sdk.Msg = IssueMsg{}

type IssueMsg struct {
	From        sdk.AccAddress `json:"from"`
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	TotalSupply int64          `json:"total_supply"`
	Mintable    bool           `json:"mintable"`
}

func NewIssueMsg(from sdk.AccAddress, name, symbol string, supply int64, mintable bool) IssueMsg {
	return IssueMsg{
		From:        from,
		Name:        name,
		Symbol:      symbol,
		TotalSupply: supply,
		Mintable:    mintable,
	}
}

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg IssueMsg) ValidateBasic() sdk.Error {
	if msg.From == nil {
		return sdk.ErrInvalidAddress("sender address cannot be empty")
	}

	if err := types.ValidateIssueSymbol(msg.Symbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	if len(msg.Name) == 0 || len(msg.Name) > maxTokenNameLength {
		return sdk.ErrInvalidCoins(fmt.Sprintf("token name should have 1 ~ %v characters", maxTokenNameLength))
	}

	if msg.TotalSupply <= 0 || msg.TotalSupply > types.TokenMaxTotalSupply {
		return sdk.ErrInvalidCoins("total supply should be less than or equal to " + string(types.TokenMaxTotalSupply))
	}

	return nil
}

// Implements IssueMsg.
func (msg IssueMsg) Route() string                { return Route }
func (msg IssueMsg) Type() string                 { return IssueMsgType }
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

	if sdk.IsUpgrade(upgrade.BEP8) && types.IsValidMiniTokenSymbol(msg.Symbol) {
		if msg.Amount < types.MiniTokenMinExecutionAmount {
			return sdk.ErrInvalidCoins(fmt.Sprintf("mint amount should be no less than %d", types.MiniTokenMinExecutionAmount))
		}
		return nil
	}

	// if BEP8 not upgraded, we rely on `ValidateTokenSymbol` rejecting the MiniToken.
	if err := types.ValidateTokenSymbol(msg.Symbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	if msg.Symbol == types.NativeTokenSymbol {
		return sdk.ErrInvalidCoins(fmt.Sprintf("cannot mint native token"))
	}

	// handler will check:  msg.Amount + token.TotalSupply <= types.MaxTotalSupply
	if msg.Amount <= 0 || msg.Amount > types.TokenMaxTotalSupply {
		return sdk.ErrInvalidCoins("total supply should be less than or equal to " + string(types.TokenMaxTotalSupply))
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
