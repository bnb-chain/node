package minitokens

import (
	"github.com/binance-chain/node/plugins/minitokens/issue"
	"github.com/binance-chain/node/plugins/minitokens/seturi"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(issue.IssueMsg{}, "minitokens/IssueMsg", nil)
	cdc.RegisterConcrete(seturi.SetURIMsg{}, "minitokens/SetURIMsg", nil)
}
