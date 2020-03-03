package bridge

import (
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(TransferMsg{}, "bridge/TransferMsg", nil)
	cdc.RegisterConcrete(TimeoutMsg{}, "bridge/TimeoutMsg", nil)
}
