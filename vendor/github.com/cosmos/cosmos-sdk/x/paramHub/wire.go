package paramHub

import (
	"github.com/tendermint/go-amino"

	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

// Register concrete types on wire codec
func RegisterWire(cdc *amino.Codec) {
	cdc.RegisterInterface((*types.FeeParam)(nil), nil)
	cdc.RegisterInterface((*types.MsgFeeParams)(nil), nil)
	cdc.RegisterConcrete(&types.FixedFeeParams{}, "params/FixedFeeParams", nil)
	cdc.RegisterConcrete(&types.TransferFeeParam{}, "params/TransferFeeParams", nil)
	cdc.RegisterConcrete(&types.DexFeeParam{}, "params/DexFeeParam", nil)
	cdc.RegisterInterface((*types.SCParam)(nil), nil)
}
