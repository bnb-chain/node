package timelock

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TimeLockRecord struct {
	Id          int64     `json:"id"`
	Description string    `json:"description"`
	Amount      sdk.Coins `json:"amount"`
	LockTime    time.Time `json:"lock_time"`
}
