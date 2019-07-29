package swap

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/tendermint/tendermint/crypto/tmhash"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewHandler creates a set account flags handler
func NewHandler(kp Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case HashTimerLockTransferMsg:
			return handleHashTimerLockTransfer(ctx, kp, msg)
		case ClaimHashTimerLockMsg:
			return handleClaimHashTimerLock(ctx, kp, msg)
		case RefundLockedAssetMsg:
			return handleRefundLockedAsset(ctx, kp, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleHashTimerLockTransfer(ctx sdk.Context, kp Keeper, msg HashTimerLockTransferMsg) sdk.Result {
	swap := &AtomicSwap{
		From:             msg.From,
		To:               msg.To,
		OutAmount:        msg.OutAmount,
		InAmount:         msg.InAmount,
		ToOnOtherChain:   msg.ToOnOtherChain,
		RandomNumberHash: msg.RandomNumberHash,
		RandomNumber:     nil,
		Timestamp:        msg.Timestamp,
		ExpireHeight:     ctx.BlockHeight() + int64(msg.TimeSpan),
		ClosedTime:		  0,
		Status:           Open,
	}
	err := kp.SaveSwap(ctx, swap);
	if err != nil {
		return err.Result()
	}

	tag, err := kp.ck.SendCoins(ctx, msg.From, AtomicSwapCoinsAccAddr, sdk.Coins{msg.OutAmount})
	if err != nil {
		return err.Result()
	}

	return sdk.Result{Tags: tag}
}

func handleClaimHashTimerLock(ctx sdk.Context, kp Keeper, msg ClaimHashTimerLockMsg) sdk.Result {
	swap := kp.QuerySwap(ctx, msg.RandomNumberHash)
	if swap == nil {
		return ErrNonExistRandomNumberHash(fmt.Sprintf("Non-exist random number hash: %v", msg.RandomNumberHash)).Result()
	}
	if swap.ExpireHeight >= ctx.BlockHeight() {
		return ErrClaimExpiredSwap(fmt.Sprintf("Swap is expired, expired height %d", swap.ExpireHeight)).Result()
	}

	randomNumberAndTimestamp := make([]byte, RandomNumberLength + 8)
	copy(randomNumberAndTimestamp[:RandomNumberLength], msg.RandomNumber)
	binary.BigEndian.PutUint64(randomNumberAndTimestamp[RandomNumberLength:], swap.Timestamp)
	if !bytes.Equal(tmhash.Sum(randomNumberAndTimestamp), msg.RandomNumberHash) {
		return ErrMismatchedRandomNumber(fmt.Sprintf("Mismatched random number")).Result()
	}

	tag, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.To, sdk.Coins{swap.OutAmount})
	if err != nil {
		return err.Result()
	}
	if ctx.IsDeliverTx() {
		kp.addrPool.AddAddrs([]sdk.AccAddress{swap.To})
	}

	ctx.BlockHeader().Time.Unix()

	swap.RandomNumber = msg.RandomNumber
	swap.Status = Completed
	swap.ClosedTime = ctx.BlockHeader().Time.Unix()
	err = kp.UpdateSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{Tags: tag}
}

func handleRefundLockedAsset(ctx sdk.Context, kp Keeper, msg RefundLockedAssetMsg) sdk.Result {
	swap := kp.QuerySwap(ctx, msg.RandomNumberHash)
	if swap == nil {
		return ErrNonExistRandomNumberHash(fmt.Sprintf("Non-exist random number hash: %v", msg.RandomNumberHash)).Result()
	}
	if swap.ExpireHeight < ctx.BlockHeight() {
		return ErrRefundUnexpiredSwap(fmt.Sprintf("Expire height (%d) is still not reached", swap.ExpireHeight)).Result()
	}

	tag, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.From, sdk.Coins{swap.OutAmount})
	if err != nil {
		return err.Result()
	}
	if ctx.IsDeliverTx(){
		kp.addrPool.AddAddrs([]sdk.AccAddress{swap.From})
	}

	swap.Status = Expired
	swap.ClosedTime = ctx.BlockHeader().Time.Unix()
	err = kp.UpdateSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{Tags: tag}
}