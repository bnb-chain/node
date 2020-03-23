package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	BindChannel        = "bind"
	TransferOutChannel = "transferOut"
	RefundChannel      = "refund"

	BindChannelID        sdk.ChannelID = 1
	TransferOutChannelID sdk.ChannelID = 2
	RefundChannelID      sdk.ChannelID = 3

	RelayFee int64 = 1e6 // 0.01BNB

	MinTransferOutExpireTimeGap = 60 * time.Second
	MinBindExpireTimeGap        = 600 * time.Second
)
