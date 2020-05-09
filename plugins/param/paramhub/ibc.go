package paramhub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/param/types"
)

const (
	IbcChannelName = "params"
	IbcChannelId   = sdk.IbcChannelID(9)
)

func (keeper *Keeper) SaveParamChangeToIbc(ctx sdk.Context, sideChainId string, paramChange types.CSCParamChange) (seq uint64, sdkErr sdk.Error) {
	if keeper.ibcKeeper == nil {
		return 0, sdk.ErrInternal("the keeper is not prepared for side chain")
	}
	bz := paramChange.Serialize()
	return keeper.ibcKeeper.CreateIBCPackage(ctx, sideChainId, IbcChannelName, bz)
}
