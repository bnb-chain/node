package airdrop

import (
	"github.com/bnb-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(AirdropApproval{}, "airdrop/AirdropApproval", nil)
}
