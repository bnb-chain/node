package bridge

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
)

func InitPlugin(appp app.ChainApp, keeper Keeper) {
	RegisterChainId(keeper)
	RegisterChannels()

	for route, handler := range Routes(keeper) {
		appp.GetRouter().AddRoute(route, handler)
	}
}

func RegisterChainId(keeper Keeper) {
	sdk.InitCrossChainID(sdk.CrossChainID(keeper.SourceChainId))
}

func RegisterChannels() {
	// NOTE: sequence matters, do not change the sequence of channels
	err := sdk.RegisterNewCrossChainChannel(types.BindChannelName)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.BindChannelName, err.Error()))
	}
	err = sdk.RegisterNewCrossChainChannel(types.TransferOutChannelName)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.TransferOutChannelName, err.Error()))
	}
	err = sdk.RegisterNewCrossChainChannel(types.RefundChannelName)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.RefundChannelName, err.Error()))
	}
}
