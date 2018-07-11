package types

import (
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

func RegisterWire(cdc *wire.Codec) {
	// Register AppAccount
	cdc.RegisterInterface((*auth.Account)(nil), nil)
	cdc.RegisterInterface((*NamedAccount)(nil), nil)
	cdc.RegisterConcrete(&AppAccount{}, "bnbchain/Account", nil)

	cdc.RegisterConcrete(Token{}, "bnbchain/Token", nil)
}
