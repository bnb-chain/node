package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	BSCBNBDecimals int8 = 18

	BindChannel        = "bind"
	TransferOutChannel = "transferOut"
	TransferInChannel  = "transferIn"

	BindChannelID        sdk.ChannelID = 1
	TransferOutChannelID sdk.ChannelID = 2
	TransferInChannelID  sdk.ChannelID = 3

	MinTransferOutExpireTimeGap = 60 * time.Second
	MinBindExpireTimeGap        = 600 * time.Second
)
