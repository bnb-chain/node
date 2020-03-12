package bridge

import (
	"github.com/binance-chain/node/plugins/bridge/keeper"
	"github.com/binance-chain/node/plugins/bridge/types"
)

var (
	NewKeeper = keeper.NewKeeper
)

type (
	Keeper = keeper.Keeper

	TransferInMsg         = types.TransferInMsg
	TransferOutTimeoutMsg = types.TransferOutTimeoutMsg
	TransferOutMsg        = types.TransferOutMsg
	BindMsg               = types.BindMsg
)
