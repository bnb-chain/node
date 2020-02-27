package bridge

import (
	"github.com/binance-chain/node/plugins/bridge/keeper"
	"github.com/binance-chain/node/plugins/bridge/types"
)

type (
	Keeper      = keeper.Keeper
	TransferMsg = types.TransferMsg
)
