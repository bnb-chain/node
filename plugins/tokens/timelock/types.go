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

type TimeLockRecords []TimeLockRecord

func (a TimeLockRecords) Len() int {
	return len(a)
}
func (a TimeLockRecords) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a TimeLockRecords) Less(i, j int) bool {
	return a[i].Id < a[j].Id
}
