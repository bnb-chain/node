package param

import (
	"github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterInterface((*types.FeeParam)(nil), nil)
	cdc.RegisterInterface((*types.MsgFeeParams)(nil), nil)
	cdc.RegisterConcrete(&types.FixedFeeParams{}, "params/FixedFeeParams", nil)
	cdc.RegisterConcrete(&types.TransferFeeParam{}, "params/TransferFeeParams", nil)
	cdc.RegisterConcrete(&types.DexFeeParam{}, "params/DexFeeParam", nil)

	cdc.RegisterInterface((*types.SCParam)(nil), nil)
	cdc.RegisterConcrete(&types.OracleParams{}, "params/OracleParams", nil)
	cdc.RegisterConcrete(&types.StakeParams{}, "params/StakeParams", nil)
	cdc.RegisterConcrete(&types.SlashParams{}, "params/SlashParams", nil)

}
