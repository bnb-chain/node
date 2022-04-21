package oracle

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/oracle/types"
)

// Register concrete types on codec codec
func RegisterWire(cdc *codec.Codec) {
	cdc.RegisterConcrete(Claim{}, "oracle/Claim", nil)
	cdc.RegisterConcrete(Prophecy{}, "oracle/Prophecy", nil)
	cdc.RegisterConcrete(Status{}, "oracle/Status", nil)
	cdc.RegisterConcrete(DBProphecy{}, "oracle/DBProphecy", nil)
	cdc.RegisterConcrete(ClaimMsg{}, "oracle/ClaimMsg", nil)
	cdc.RegisterConcrete(&types.Params{}, "params/OracleParamSet", nil)
}
