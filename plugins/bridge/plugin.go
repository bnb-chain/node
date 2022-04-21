package bridge

import (
	app "github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/plugins/bridge/types"
)

func InitPlugin(chainApp app.ChainApp, keeper Keeper) {
	for route, handler := range Routes(keeper) {
		chainApp.GetRouter().AddRoute(route, handler)
	}

	RegisterCrossApps(keeper)
}

func RegisterCrossApps(keeper Keeper) {
	updateBindApp := NewBindApp(keeper)
	err := keeper.ScKeeper.RegisterChannel(types.BindChannel, types.BindChannelID, updateBindApp)
	if err != nil {
		panic(err)
	}

	transferOutRefundApp := NewTransferOutApp(keeper)
	err = keeper.ScKeeper.RegisterChannel(types.TransferOutChannel, types.TransferOutChannelID, transferOutRefundApp)
	if err != nil {
		panic(err)
	}

	transferInApp := NewTransferInApp(keeper)
	err = keeper.ScKeeper.RegisterChannel(types.TransferInChannel, types.TransferInChannelID, transferInApp)
	if err != nil {
		panic(err)
	}

	mirrorApp := NewMirrorApp(keeper)
	err = keeper.ScKeeper.RegisterChannel(types.MirrorChannel, types.MirrorChannelID, mirrorApp)
	if err != nil {
		panic(err)
	}

	mirrorSyncApp := NewMirrorSyncApp(keeper)
	err = keeper.ScKeeper.RegisterChannel(types.MirrorSyncChannel, types.MirrorSyncChannelID, mirrorSyncApp)
	if err != nil {
		panic(err)
	}
}
