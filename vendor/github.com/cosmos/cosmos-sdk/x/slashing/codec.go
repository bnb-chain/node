package slashing

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// Register concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgUnjail{}, "cosmos-sdk/MsgUnjail", nil)
	cdc.RegisterConcrete(MsgSideChainUnjail{}, "cosmos-sdk/MsgSideChainUnjail", nil)
	cdc.RegisterConcrete(MsgBscSubmitEvidence{}, "cosmos-sdk/MsgBscSubmitEvidence", nil)
	cdc.RegisterConcrete(&Params{}, "params/SlashParamSet", nil)
}

// generic sealed codec to be used throughout sdk
var MsgCdc *codec.Codec

func init() {
	cdc := codec.New()
	RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	MsgCdc = cdc.Seal()
}
