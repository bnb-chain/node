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
	sdk.SetSourceChainID(sdk.CrossChainID(keeper.SourceChainId))
}

func RegisterChannels() {
	// NOTE: sequence matters, do not change the sequence of channels
	err := sdk.RegisterNewCrossChainChannel(types.BindChannel, types.BindChannelID)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.BindChannel, err.Error()))
	}
	err = sdk.RegisterNewCrossChainChannel(types.TransferOutChannel, types.TransferOutChannelID)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.TransferOutChannel, err.Error()))
	}
	err = sdk.RegisterNewCrossChainChannel(types.RefundChannel, types.RefundChannelID)
	if err != nil {
		panic(fmt.Sprintf("register channel error, channel=%s, err=%s", types.RefundChannel, err.Error()))
	}
}
