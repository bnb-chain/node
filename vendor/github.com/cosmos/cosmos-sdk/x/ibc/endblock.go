package ibc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx sdk.Context, keeper Keeper) {
	if len(keeper.packageCollector.collectedPackages) == 0 {
		return
	}
	var attributes []sdk.Attribute
	for _, ibcPackageRecord := range keeper.packageCollector.collectedPackages {
		attributes = append(attributes,
			sdk.NewAttribute(ibcPackageInfoAttributeKey,
				buildIBCPackageAttributeValue(ibcPackageRecord.destChainID, ibcPackageRecord.channelID, ibcPackageRecord.sequence)))
	}
	keeper.packageCollector.collectedPackages = keeper.packageCollector.collectedPackages[:0]
	event := sdk.NewEvent(ibcEventType, attributes...)
	ctx.EventManager().EmitEvent(event)
}
