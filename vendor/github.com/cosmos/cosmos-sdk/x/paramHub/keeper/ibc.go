package keeper

import (
	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

const (
	ChannelName = "params"
	ChannelId   = sdk.ChannelID(9)
)

func (keeper *Keeper) SaveParamChangeToIbc(ctx sdk.Context, sideChainId string, paramChange types.CSCParamChange) (seq uint64, sdkErr sdk.Error) {
	if keeper.ibcKeeper == nil {
		return 0, sdk.ErrInternal("the keeper is not prepared for side chain")
	}
	bz, err := rlp.EncodeToBytes(&paramChange)
	if err != nil {
		return 0, sdk.ErrInternal("failed to encode paramChange")
	}
	return keeper.ibcKeeper.CreateIBCSyncPackage(ctx, sideChainId, ChannelName, bz)
}
