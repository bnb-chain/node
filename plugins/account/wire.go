package account

import (
	"github.com/binance-chain/node/plugins/account/setaccountflags"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(setaccountflags.SetAccountFlagsMsg{}, "scripts/SetAccountFlagsMsg", nil)
}
