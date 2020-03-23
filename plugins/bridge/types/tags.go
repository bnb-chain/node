package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	Action = sdk.TagAction

	ActionTransferInFailed = []byte("TransferInFailed")

	ExpireTime                 = "ExpireTime"
	BindSequence               = "BindSequence"
	UpdateBindSequence         = "UpdateBindSequence"
	TransferInSequence         = "TransferInSequence"
	TransferInRefundSequence   = "TransferInRefundSequence"
	TransferInRefundReason     = "TransferInRefundReason"
	TransferOutSequence        = "TransferOutSequence"
	TransferOutRefundSequence  = "TransferOutRefundSequence"
	TransferOutRefundReason    = "TransferOutRefundReason"
)
