package context

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// StdSignMsg is a convenience structure for passing along
// a Msg with the other requirements for a StdSignDoc before
// it is signed. For use in the CLI.
type StdSignMsg struct {
	ChainID       string    `json:"chain_id"`
	AccountNumber int64     `json:"account_number"`
	Sequence      int64     `json:"sequence"`
	Msgs          []sdk.Msg `json:"msgs"`
	Memo          string    `json:"memo"`
	Source        int64     `json:"source"`
	Data          []byte    `json:"data"`
}

// get message bytes
func (msg StdSignMsg) Bytes() []byte {
	return auth.StdSignBytes(msg.ChainID, msg.AccountNumber, msg.Sequence, msg.Msgs, msg.Memo, msg.Source, msg.Data)
}
