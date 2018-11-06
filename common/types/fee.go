package types

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type FeeDistributeType int8

const (
	FeeForProposer = FeeDistributeType(0x01)
	FeeForAll      = FeeDistributeType(0x02)
	FeeFree        = FeeDistributeType(0x03)

	serializeSeparator   = ";"
	amountDenomSeparator = ":"
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

// Any change of this method should communicate with services (query, explorer) developers
func (fee Fee) Serialize() string {
	if fee.IsEmpty() {
		return ""
	} else {
		var buffer bytes.Buffer
		for _, coin := range fee.Tokens {
			buffer.WriteString(fmt.Sprintf("%s%s%s%s", coin.Denom, amountDenomSeparator, coin.Amount.String(), serializeSeparator))
		}
		res := buffer.String()
		return res[:len(res)-1]
	}
}

func (fee Fee) String() string {
	return fee.Serialize()
}
