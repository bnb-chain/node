package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	BSCBNBDecimals int8 = 18

	BindChannel        = "bind"
	TransferOutChannel = "transferOut"
	RefundChannel      = "refund"

	BindChannelID        sdk.IbcChannelID = 1
	TransferOutChannelID sdk.IbcChannelID = 2
	RefundChannelID      sdk.IbcChannelID = 3

	MinTransferOutExpireTimeGap = 60 * time.Second
	MinBindExpireTimeGap        = 600 * time.Second
)
