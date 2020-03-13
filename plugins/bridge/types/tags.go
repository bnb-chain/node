package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	Action = sdk.TagAction

	ActionBind               = []byte("bind")
	ActionUpdateBind         = []byte("update-bind")
	ActionTransferOut        = []byte("transfer-out")
	ActionTransferIn         = []byte("transfer-in")
	ActionTransferInTimeOut  = []byte("transfer-in-timeout")
	ActionTransferOutTimeout = []byte("transfer-out-timeout")

	ExpireTime                 = "expire-time"
	BindSequence               = "bind-sequence"
	UpdateBindSequence         = "update-bind-sequence"
	TransferInSequence         = "transfer-in-sequence"
	TransferInTimeoutSequence  = "transfer-in-timeout-sequence"
	TransferOutSequence        = "transfer-out-sequence"
	TransferOutTimeoutSequence = "transfer-out-timeout-sequence"
)
