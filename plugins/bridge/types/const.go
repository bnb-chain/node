package types

import (
	"time"
)

const (
	RelayFee int64 = 1e6 // 0.01BNB

	MinTransferOutExpireTimeGap = 60 * time.Second
	MinBindExpireTimeGap        = 600 * time.Second
)
