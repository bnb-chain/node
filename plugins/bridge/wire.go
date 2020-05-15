package bridge

import (
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(BindMsg{}, "bridge/BindMsg", nil)
	cdc.RegisterConcrete(TransferOutMsg{}, "bridge/TransferOutMsg", nil)
}
