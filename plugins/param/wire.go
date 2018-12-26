package param

import (
	"github.com/BiJie/BinanceChain/plugins/param/types"
	"github.com/BiJie/BinanceChain/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterInterface((*types.FeeParam)(nil), nil)
	cdc.RegisterConcrete(&types.FixedFeeParams{}, "params/FixedFeeParams", nil)
	cdc.RegisterConcrete(&types.DexFeeParam{}, "params/DexFeeParam", nil)
}
