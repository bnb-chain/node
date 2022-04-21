package sidechain

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
)

// Default parameter namespace
const DefaultParamspace = "sidechain"

var (
	KeyBscSideChainId = []byte("BscSideChainId")
)

// ParamTypeTable for sidechain module
func ParamTypeTable() params.TypeTable {
	return params.NewTypeTable().RegisterParamSet(&Params{})
}

type Params struct {
	BscSideChainId string `json:"bsc_side_chain_id"`
}

// Implements params.ParamStruct
func (p *Params) KeyValuePairs() params.KeyValuePairs {
	return params.KeyValuePairs{
		{KeyBscSideChainId, &p.BscSideChainId},
	}
}

// Default parameters used by Cosmos Hub
func DefaultParams() Params {
	return Params{
		BscSideChainId: "bsc",
	}
}

func (k Keeper) BscSideChainId(ctx sdk.Context) (sideChainId string) {
	k.paramspace.Get(ctx, KeyBscSideChainId, &sideChainId)
	return
}

// set the params
func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	k.paramspace.SetParamSet(ctx, &params)
}
