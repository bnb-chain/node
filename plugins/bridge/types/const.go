package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	BindChannel        = "bind"
	TransferOutChannel = "transferOut"
	RefundChannel      = "refund"

	BindChannelID        sdk.IbcChannelID = 1
	TransferOutChannelID sdk.IbcChannelID = 2
	RefundChannelID      sdk.IbcChannelID = 3

	RelayFee int64 = 1e6 // 0.01BNB

	MinTransferOutExpireTimeGap = 60 * time.Second
	MinBindExpireTimeGap        = 600 * time.Second
)
