package timelock

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TimeLockRecord struct {
	Id          int64
	Description string
	Amount      sdk.Coins
	LockTime    time.Time
}
