package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
)


func RegisterTypes(cdc *wire.Codec) {
	// Register AppAccount
	cdc.RegisterInterface((*sdk.Account)(nil), nil)
	cdc.RegisterConcrete(&AppAccount{}, "bnbchain/Account", nil)

	cdc.RegisterConcrete(Token{}, "bnbchain/Token", nil)
	cdc.RegisterConcrete(Number{}, "bnbchain/Number", nil)
}
