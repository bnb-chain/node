package tokens

import (
	"encoding/json"
	"fmt"

	"github.com/BiJie/BinanceChain/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type IssueMsg struct {
	Owner sdk.Address `json:"owner"`
	Token types.Token `json:"token"`
}

// NewIssueMsg - construct arbitrary multi-in, multi-out send msg.
func NewIssueMsg(owner sdk.Address, token types.Token) IssueMsg {
	return IssueMsg{Owner: owner, Token: token}
}

func (msg IssueMsg) Type() string { return "tokens" }

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg IssueMsg) ValidateBasic() sdk.Error {
	err := msg.Token.Validate()
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	return nil
}

func (msg IssueMsg) String() string {
	return fmt.Sprintf("IssueMsg{%v#%v}", msg.Owner, msg.Token)
}

func (msg IssueMsg) Get(key interface{}) (value interface{}) {
	return nil
}

func (msg IssueMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

// Implements Msg.
func (msg IssueMsg) GetSigners() []sdk.Address {
	return []sdk.Address{msg.Owner}
}
