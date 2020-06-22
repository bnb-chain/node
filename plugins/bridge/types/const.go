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

	BindChannelID        sdk.IbcChannelID = 1
	TransferOutChannelID sdk.IbcChannelID = 2
	TransferInChannelID  sdk.IbcChannelID = 3

	MinTransferOutExpireTimeGap = 60 * time.Second
	MinBindExpireTimeGap        = 600 * time.Second
)
