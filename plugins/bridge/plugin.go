package bridge

import (
	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
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
}
