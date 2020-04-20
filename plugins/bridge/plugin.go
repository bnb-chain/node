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

	InitOracle(keeper)
}

func RegisterChannels(keeper Keeper) {
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

func InitOracle(keeper Keeper) {
	skipSequenceHooks := NewSkipSequenceClaimHooks(keeper)
	err := keeper.OracleKeeper.RegisterClaimType(types.ClaimTypeSkipSequence, types.ClaimTypeSkipSequenceName, skipSequenceHooks)
	if err != nil {
		panic(err)
	}

	updateBindClaimHooks := NewUpdateBindClaimHooks(keeper)
	err = keeper.OracleKeeper.RegisterClaimType(types.ClaimTypeUpdateBind, types.ClaimTypeUpdateBindName, updateBindClaimHooks)
	if err != nil {
		panic(err)
	}

	updateTransferOutClaimHooks := NewUpdateTransferOutClaimHooks(keeper)
	err = keeper.OracleKeeper.RegisterClaimType(types.ClaimTypeUpdateTransferOut, types.ClaimTypeUpdateTransferOutName, updateTransferOutClaimHooks)
	if err != nil {
		panic(err)
	}

	transferInClaimHooks := NewTransferInClaimHooks(keeper)
	err = keeper.OracleKeeper.RegisterClaimType(types.ClaimTypeTransferIn, types.ClaimTypeTransferInName, transferInClaimHooks)
	if err != nil {
		panic(err)
	}
}
