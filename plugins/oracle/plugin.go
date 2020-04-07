package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/oracle/types"
)

func InitPlugin(keeper Keeper) {
	RegisterUpgradeBeginBlocker(keeper)
}

func RegisterUpgradeBeginBlocker(keeper Keeper) {
	upgrade.Mgr.RegisterBeginBlocker(upgrade.BSCUpgrade, func(ctx sdk.Context) {
		keeper.SetProphecyParams(ctx, types.ProphecyParams{ConsensusNeeded: types.DefaultConsensusNeeded})
	})
}
