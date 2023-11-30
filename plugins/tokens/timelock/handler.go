package timelock

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case TimeLockMsg:
			if sdk.IsUpgrade(sdk.FirstSunsetFork) {
				return sdk.ErrMsgNotSupported("").Result()
			}
			return handleTimeLock(ctx, keeper, msg)
		case TimeUnlockMsg:
			return handleTimeUnlock(ctx, keeper, msg)
		case TimeRelockMsg:
			if sdk.IsUpgrade(sdk.FirstSunsetFork) {
				return sdk.ErrMsgNotSupported("").Result()
			}
			return handleTimeRelock(ctx, keeper, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized time lock message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleTimeLock(ctx sdk.Context, keeper Keeper, msg TimeLockMsg) sdk.Result {
	record, err := keeper.TimeLock(ctx, msg.From, msg.Description, msg.Amount, time.Unix(msg.LockTime, 0))
	if err != nil {
		return err.Result()
	}

	timeLockId := []byte(fmt.Sprintf("%d", record.Id))
	return sdk.Result{
		Data: timeLockId,
	}
}

func handleTimeRelock(ctx sdk.Context, keeper Keeper, msg TimeRelockMsg) sdk.Result {
	newRecord := TimeLockRecord{
		Description: msg.Description,
		Amount:      msg.Amount,
		LockTime:    time.Unix(msg.LockTime, 0),
	}

	err := keeper.TimeRelock(ctx, msg.From, msg.Id, newRecord)
	if err != nil {
		return err.Result()
	}

	timeLockId := []byte(fmt.Sprintf("%d", msg.Id))
	return sdk.Result{
		Data: timeLockId,
	}
}

func handleTimeUnlock(ctx sdk.Context, keeper Keeper, msg TimeUnlockMsg) sdk.Result {
	err := keeper.TimeUnlock(ctx, msg.From, msg.Id)
	if err != nil {
		return err.Result()
	}

	timeLockId := []byte(fmt.Sprintf("%d", msg.Id))
	return sdk.Result{
		Data: timeLockId,
	}
}
