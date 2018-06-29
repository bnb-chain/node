package dex

import (
	"github.com/cosmos/cosmos-sdk/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(MakeOfferMsg{}, "cool/MakeOffer", nil)
	cdc.RegisterConcrete(FillOfferMsg{}, "cool/FillOffer", nil)
	cdc.RegisterConcrete(CancelOfferMsg{}, "cool/CancelOffer", nil)
}
