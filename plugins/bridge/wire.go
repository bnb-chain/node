package bridge

import (
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(TransferInMsg{}, "bridge/TransferInMsg", nil)
	cdc.RegisterConcrete(TimeoutMsg{}, "bridge/TimeoutMsg", nil)
	cdc.RegisterConcrete(BindMsg{}, "bridge/BindMsg", nil)
	cdc.RegisterConcrete(TransferOutMsg{}, "bridge/TransferOutMsg", nil)
}
