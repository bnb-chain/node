package ibc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type packageRecord struct {
	destChainID sdk.ChainID
	channelID   sdk.ChannelID
	sequence    uint64
}

type packageCollector struct {
	collectedPackages []packageRecord
}

func newPackageCollector() *packageCollector {
	return &packageCollector{
		collectedPackages: nil,
	}
}
