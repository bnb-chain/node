package issue

import (
	"encoding/json"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route  = "tokens/issue"
const Route = "tokensIssue"

var _ sdk.Msg = Msg{}

type Msg struct {
	From        sdk.AccAddress `json:"from"`
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	TotalSupply int64          `json:"total_supply"`
}

func NewMsg(from sdk.AccAddress, name, symbol string, supply int64) Msg {
	return Msg{
		From:        from,
		Name:        name,
		Symbol:      symbol,
		TotalSupply: supply,
	}
}

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg Msg) ValidateBasic() sdk.Error {
	if msg.From == nil {
		return sdk.ErrInvalidAddress("sender address cannot be empty")
	}

	if err := types.ValidateSymbol(msg.Symbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	if len(msg.Name) == 0 || len(msg.Name) > 20 {
		return sdk.ErrInvalidCoins("token name should have 1~20 characters")
	}

	if msg.TotalSupply <= 0 || msg.TotalSupply > types.MaxTotalSupply {
		return sdk.ErrInvalidCoins("total supply should be <= " + string(types.MaxTotalSupply/int64(math.Pow10(int(types.Decimals)))))
	}

	return nil
}

// Implements Msg.
func (msg Msg) Type() string                            { return Route }
func (msg Msg) String() string                          { return fmt.Sprintf("IssueMsg{%#v}", msg) }
func (msg Msg) Get(key interface{}) (value interface{}) { return nil }
func (msg Msg) GetSigners() []sdk.AccAddress            { return []sdk.AccAddress{msg.From} }

func (msg Msg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
