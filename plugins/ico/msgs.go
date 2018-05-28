package ico

import (
	"fmt"
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)


type IssueMsg struct {
	Banker  sdk.Address `json:"banker"`
	Coin    sdk.Coin	`json:"coin"`
}

// NewIssueMsg - construct arbitrary multi-in, multi-out send msg.
func NewIssueMsg(banker sdk.Address, coin sdk.Coin) IssueMsg {
	return IssueMsg{Banker: banker, Coin: coin}
}

// Implements Msg.
func (msg IssueMsg) Type() string { return "ico" }

// Implements Msg.
func (msg IssueMsg) ValidateBasic() sdk.Error {
	// TODO
	return nil
}

func (msg IssueMsg) String() string {
	return fmt.Sprintf("IssueMsg{%v#%v}", msg.Banker, msg.Coin)
}

// Implements Msg.
func (msg IssueMsg) Get(key interface{}) (value interface{}) {
	return nil
}

// Implements Msg.
func (msg IssueMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

// Implements Msg.
func (msg IssueMsg) GetSigners() []sdk.Address {
	return []sdk.Address{msg.Banker}
}