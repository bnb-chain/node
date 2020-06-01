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
	UnbindMsg      = types.UnbindMsg

	SkipSequenceClaim      = types.SkipSequenceClaim
	UpdateBindClaim        = types.UpdateBindClaim
	TransferOutRefundClaim = types.TransferOutRefundClaim
	TransferInClaim        = types.TransferInClaim
)

const (
	BindRelayFeeName   = types.BindRelayFeeName
	UnbindRelayFeeName   = types.UnbindRelayFeeName
	TransferOutFeeName = types.TransferOutFeeName
)
