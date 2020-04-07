package bridge

import (
	"fmt"

	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
)

func InitPlugin(appp app.ChainApp, keeper Keeper) {
	RegisterChannels(keeper)

	for route, handler := range Routes(keeper) {
		appp.GetRouter().AddRoute(route, handler)
	}
}

func RegisterChannels(keeper Keeper) {
	// NOTE: sequence matters, do not change the sequence of channels
	err := keeper.IbcKeeper.RegisterChannel(types.BindChannel, types.BindChannelID)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.BindChannel, err.Error()))
	}
	err = keeper.IbcKeeper.RegisterChannel(types.TransferOutChannel, types.TransferOutChannelID)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.TransferOutChannel, err.Error()))
	}
	err = keeper.IbcKeeper.RegisterChannel(types.RefundChannel, types.RefundChannelID)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.RefundChannel, err.Error()))
	}
}
