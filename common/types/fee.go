package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type FeeDistributeType int8

const (
	FeeForProposer = FeeDistributeType(0x01)
	FeeForAll      = FeeDistributeType(0x02)
	FeeFree        = FeeDistributeType(0x03)
)

type Fee struct {
	Tokens sdk.Coins
	Type   FeeDistributeType
}

func NewFee(tokens sdk.Coins, distributeType FeeDistributeType) Fee {
	return Fee{
		Tokens: tokens,
		Type:   distributeType,
	}
}

func (fee Fee) IsEmpty() bool {
	return fee.Tokens == nil
}
