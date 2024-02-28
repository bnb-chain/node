package migrate

import (
	"github.com/bnb-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(ValidatorOwnerShip{}, "migrate/ValidatorOwnerShip", nil)
}
