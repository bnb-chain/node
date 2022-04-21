package keeper

import (
	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

const ChannelName = "stake"
const ChannelId = sdk.ChannelID(8)

func (k Keeper) SaveValidatorSetToIbc(ctx sdk.Context, sideChainId string, ibcPackage types.IbcValidatorSetPackage) (seq uint64, sdkErr sdk.Error) {
	if k.ibcKeeper == nil {
		return 0, sdk.ErrInternal("the keeper is not prepared for side chain")
	}
	bz, err := rlp.EncodeToBytes(ibcPackage)
	if err != nil {
		return 0, sdk.ErrInternal("failed to encode IbcValidatorSetPackage")
	}
	return k.ibcKeeper.CreateIBCSyncPackage(ctx, sideChainId, ChannelName, bz)
}
