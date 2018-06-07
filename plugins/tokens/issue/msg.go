package issue

import (
	"encoding/json"
	"fmt"

	"github.com/BiJie/BinanceChain/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route  = "tokens/issue"
const Route  = "tokensIssue"

var _ sdk.Msg = (*Msg)(nil)

type Msg struct {
	Owner sdk.Address `json:"owner"`
	Token types.Token `json:"token"`
}

func NewMsg(owner sdk.Address, token types.Token) Msg {
	return Msg{Owner: owner, Token: token}
}

func (msg Msg) Type() string { return Route }

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg Msg) ValidateBasic() sdk.Error {
	err := msg.Token.Validate()
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("IssueMsg{%v#%v}", msg.Owner, msg.Token)
}

func (msg Msg) Get(key interface{}) (value interface{}) {
	return nil
}

func (msg Msg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

// Implements Msg.
func (msg Msg) GetSigners() []sdk.Address {
	return []sdk.Address{msg.Owner}
}
