package ibc

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

func RegisterWire(cdc *codec.Codec) {
	cdc.RegisterConcrete(&Params{}, "params/IbcParamSet", nil)
}
