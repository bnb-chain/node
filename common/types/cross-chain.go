package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// supported destination chain
	BSCChain = "bsc"

	// supported channel
	BindChannel        = "bind"
	TransferOutChannel = "transferOut"
	RefundChannel      = "refund"
	StakingChannel     = "staking"

	BindChannelID        sdk.ChannelID = 1
	TransferOutChannelID sdk.ChannelID = 2
	RefundChannelID      sdk.ChannelID = 3
	StakingChannelID     sdk.ChannelID = 4
)
