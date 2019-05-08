package timelock

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func handleTimeLock(ctx sdk.Context, keeper Keeper, msg TimeLockMsg) sdk.Result {
	lockTime := time.Unix(msg.LockTime, 0)
	if lockTime.Before(ctx.BlockHeader().Time) {
		return ErrInvalidLockTime(DefaultCodespace, fmt.Sprintf("lock time(%s) should after now(%s)",
			lockTime.String(), ctx.BlockHeader().Time.String())).Result()
	}

	err := keeper.TimeLock(ctx, msg.From, msg.Description, msg.Amount, lockTime)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

func handleTimeRelock(ctx sdk.Context, keeper Keeper, msg TimeRelockMsg) sdk.Result {
	if msg.LockTime != 0 {
		lockTime := time.Unix(msg.LockTime, 0)
		if lockTime.Before(ctx.BlockHeader().Time) {
			return ErrInvalidLockTime(DefaultCodespace, fmt.Sprintf("lock time(%s) should after now(%s)",
				lockTime.String(), ctx.BlockHeader().Time.String())).Result()
		}
	}

	newRecord := TimeLockRecord{
		Description: msg.Description,
		Amount:      msg.Amount,
		LockTime:    time.Unix(msg.LockTime, 0),
	}

	err := keeper.TimeRelock(ctx, msg.From, msg.Id, newRecord)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

func handleTimeUnlock(ctx sdk.Context, keeper Keeper, msg TimeUnlockMsg) sdk.Result {
	err := keeper.TimeUnlock(ctx, msg.From, msg.Id)
	if err != nil {
		return err.Result()
	}
	return sdk.Result{}
}
