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

func (fee *Fee) AddFee(other Fee) {
	if other.IsEmpty() {
		return
	}

	if fee.Tokens == nil {
		fee.Tokens = other.Tokens
		fee.Type = other.Type
	} else {
		fee.Tokens = fee.Tokens.Plus(other.Tokens)
		if other.Type == FeeForAll {
			fee.Type = FeeForAll
		}
	}
}

func (fee Fee) IsEmpty() bool {
	return fee.Tokens == nil || fee.Tokens.IsEqual(sdk.Coins{})
}
