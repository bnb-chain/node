package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	Action = sdk.TagAction

	ActionBind        = []byte("bind")
	ActionUpdateBind  = []byte("update-bind")
	ActionTransferOut = []byte("transfer-out")
	ActionTransferIn  = []byte("transfer-in")
	ActionTimeout     = []byte("timeout")

	ExpireTime = "expire-time"
)
