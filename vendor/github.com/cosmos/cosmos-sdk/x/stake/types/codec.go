package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// Register concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgCreateValidator{}, "cosmos-sdk/MsgCreateValidator", nil)
	cdc.RegisterConcrete(MsgRemoveValidator{}, "cosmos-sdk/MsgRemoveValidator", nil)
	cdc.RegisterConcrete(MsgCreateValidatorProposal{}, "cosmos-sdk/MsgCreateValidatorProposal", nil)
	cdc.RegisterConcrete(MsgEditValidator{}, "cosmos-sdk/MsgEditValidator", nil)
	cdc.RegisterConcrete(MsgDelegate{}, "cosmos-sdk/MsgDelegate", nil)
	cdc.RegisterConcrete(MsgBeginUnbonding{}, "cosmos-sdk/BeginUnbonding", nil)
	cdc.RegisterConcrete(MsgBeginRedelegate{}, "cosmos-sdk/BeginRedelegate", nil)

	cdc.RegisterConcrete(MsgCreateSideChainValidator{}, "cosmos-sdk/MsgCreateSideChainValidator", nil)
	cdc.RegisterConcrete(MsgEditSideChainValidator{}, "cosmos-sdk/MsgEditSideChainValidator", nil)
	cdc.RegisterConcrete(MsgSideChainDelegate{}, "cosmos-sdk/MsgSideChainDelegate", nil)
	cdc.RegisterConcrete(MsgSideChainRedelegate{}, "cosmos-sdk/MsgSideChainRedelegate", nil)
	cdc.RegisterConcrete(MsgSideChainUndelegate{}, "cosmos-sdk/MsgSideChainUndelegate", nil)

	cdc.RegisterConcrete(&Params{}, "params/StakeParamSet", nil)
}

// generic sealed codec to be used throughout sdk
var MsgCdc *codec.Codec

func init() {
	cdc := codec.New()
	RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	MsgCdc = cdc.Seal()
}
