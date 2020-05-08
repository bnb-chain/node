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

	TransferOutMsg = types.TransferOutMsg
	BindMsg        = types.BindMsg

	SkipSequenceClaim      = types.SkipSequenceClaim
	UpdateBindClaim        = types.UpdateBindClaim
	TransferOutRefundClaim = types.TransferOutRefundClaim
	TransferInClaim        = types.TransferInClaim
)

const (
	BindRelayFeeName   = types.BindRelayFeeName
	TransferOutFeeName = types.TransferOutFeeName
)
