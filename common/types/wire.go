package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/wire"
)

func RegisterWire(cdc *wire.Codec) {
	// Register AppAccount
	cdc.RegisterInterface((*sdk.Account)(nil), nil)
	cdc.RegisterInterface((*NamedAccount)(nil), nil)
	cdc.RegisterInterface((*IToken)(nil), nil)

	cdc.RegisterConcrete(&AppAccount{}, "bnbchain/Account", nil)

	cdc.RegisterConcrete(&Token{}, "bnbchain/Token", nil)
	cdc.RegisterConcrete(&MiniToken{}, "bnbchain/MiniToken", nil)
}
