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

	BindChannelID        sdk.ChannelID = 1
	TransferOutChannelID sdk.ChannelID = 2
	RefundChannelID      sdk.ChannelID = 3
)

func IsDestChainRegistered(chainName string) bool {
	if chainName == BSCChain {
		return true
	}
	return false
}
