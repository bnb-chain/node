package tokens

import (
	"github.com/cosmos/cosmos-sdk/wire"
)

// Register concrete types on wire codec
func RegisterTypes(cdc *wire.Codec) {
	cdc.RegisterConcrete(IssueMsg{}, "tokens/IssueMsg", nil)
}