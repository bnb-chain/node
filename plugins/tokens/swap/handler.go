package swap

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewHandler(kp Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case HashTimerLockTransferMsg:
			return handleHashTimerLockTransfer(ctx, kp, msg)
		case ClaimHashTimerLockMsg:
			return handleClaimHashTimerLock(ctx, kp, msg)
		case RefundHashTimerLockMsg:
			return handleRefundHashTimerLock(ctx, kp, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleHashTimerLockTransfer(ctx sdk.Context, kp Keeper, msg HashTimerLockTransferMsg) sdk.Result {
	if msg.Timestamp < ctx.BlockHeader().Time.Unix()-TwoHour || msg.Timestamp > ctx.BlockHeader().Time.Unix()+OneHour {
		return ErrCodeInvalidTimestamp(fmt.Sprintf("The timestamp (%d) should not be one hour ahead or two hours behind block time (%d)", msg.Timestamp, ctx.BlockHeader().Time.Unix())).Result()
	}

	tags, err := kp.ck.SendCoins(ctx, msg.From, AtomicSwapCoinsAccAddr, sdk.Coins{msg.OutAmount})
	if err != nil {
		return err.Result()
	}

	swap := kp.QuerySwap(ctx, msg.RandomNumberHash)
	if swap != nil {
		if swap.CrossChain || msg.CrossChain {
			return ErrCodeInvalidSingleChainSwap("Both the swap and msg should be single chain mode").Result()
		}
		if swap.Status != Open {
			return ErrCodeInvalidSingleChainSwap("Swap status is not open").Result()
		}
		if ctx.BlockHeight() >= swap.ExpireHeight {
			return ErrCodeInvalidSingleChainSwap("The swap expire height is passed").Result()
		}
		if !bytes.Equal(swap.From, msg.To) || !bytes.Equal(swap.To, msg.From) {
			return ErrCodeInvalidSingleChainSwap("Addresses don't match").Result()
		}
		if !swap.InAmount.IsZero() {
			return ErrCodeInvalidSingleChainSwap("The swap has already been responded").Result()
		}
		swap.InAmount = msg.OutAmount
		err := kp.UpdateSwap(ctx, swap)
		if err != nil {
			return err.Result()
		}
		return  sdk.Result{Tags: tags}
	}
	swap = &AtomicSwap{
		From:                msg.From,
		To:                  msg.To,
		OutAmount:           msg.OutAmount,
		InAmount:            sdk.Coin{},
		ExpectedIncome:      msg.ExpectedIncome,
		RecipientOtherChain: msg.RecipientOtherChain,
		RandomNumberHash:    msg.RandomNumberHash,
		RandomNumber:        nil,
		Timestamp:           msg.Timestamp,
		ExpireHeight:        ctx.BlockHeight() + int64(msg.HeightSpan),
		CrossChain:          msg.CrossChain,
		ClosedTime:          0,
		Status:              Open,
		Index:               kp.GetIndex(ctx),
	}
	err = kp.CreateSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{Tags: tags}
}

func handleClaimHashTimerLock(ctx sdk.Context, kp Keeper, msg ClaimHashTimerLockMsg) sdk.Result {
	swap := kp.QuerySwap(ctx, msg.RandomNumberHash)
	if swap == nil {
		return ErrNonExistRandomNumberHash(fmt.Sprintf("There is no swap matched with randomNumberHash %v", msg.RandomNumberHash)).Result()
	}
	if swap.Status != Open {
		return ErrUnexpectedSwapStatus(fmt.Sprintf("Expected swap status is Open, actual it is %s", swap.Status.String())).Result()
	}
	if swap.ExpireHeight <= ctx.BlockHeight() {
		return ErrClaimExpiredSwap(fmt.Sprintf("Swap is expired, expired height %d", swap.ExpireHeight)).Result()
	}

	if !bytes.Equal(CalculateRandomHash(msg.RandomNumber, swap.Timestamp), msg.RandomNumberHash) {
		return ErrMismatchedRandomNumber(fmt.Sprintf("Mismatched random number")).Result()
	}

	var err sdk.Error
	var toTags sdk.Tags
	var fromTags sdk.Tags
	if !swap.OutAmount.IsZero() {
		toTags, err = kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.To, sdk.Coins{swap.OutAmount})
		if err != nil {
			return err.Result()
		}
	}
	if !swap.InAmount.IsZero() {
		fromTags, err = kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.From, sdk.Coins{swap.InAmount})
		if err != nil {
			return err.Result()
		}
	}
	if ctx.IsDeliverTx() && kp.addrPool != nil {
		if !bytes.Equal(msg.From, swap.From) {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.From})
		}
		if !bytes.Equal(msg.From, swap.To) {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.To})
		}
	}

	swap.RandomNumber = msg.RandomNumber
	swap.Status = Completed
	swap.ClosedTime = ctx.BlockHeader().Time.Unix()
	err = kp.CloseSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}
	tags := toTags.AppendTags(fromTags)
	return sdk.Result{Tags: tags}
}

func handleRefundHashTimerLock(ctx sdk.Context, kp Keeper, msg RefundHashTimerLockMsg) sdk.Result {
	swap := kp.QuerySwap(ctx, msg.RandomNumberHash)
	if swap == nil {
		return ErrNonExistRandomNumberHash(fmt.Sprintf("There is no swap matched with randomNumberHash %v", msg.RandomNumberHash)).Result()
	}
	if swap.Status != Open {
		return ErrUnexpectedSwapStatus(fmt.Sprintf("Expected swap status is Open, actual it is %s", swap.Status.String())).Result()
	}
	if ctx.BlockHeight() < swap.ExpireHeight {
		return ErrRefundUnexpiredSwap(fmt.Sprintf("Expire height (%d) is still not reached", swap.ExpireHeight)).Result()
	}

	var err sdk.Error
	var toTags sdk.Tags
	var fromTags sdk.Tags
	if !swap.OutAmount.IsZero() {
		fromTags, err = kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.From, sdk.Coins{swap.OutAmount})
		if err != nil {
			return err.Result()
		}
	}
	if !swap.InAmount.IsZero() {
		toTags, err = kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.To, sdk.Coins{swap.InAmount})
		if err != nil {
			return err.Result()
		}
	}
	if ctx.IsDeliverTx() && kp.addrPool != nil {
		if !bytes.Equal(msg.From, swap.From) {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.From})
		}
		if !bytes.Equal(msg.From, swap.To) {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.To})
		}
	}

	swap.Status = Expired
	swap.ClosedTime = ctx.BlockHeader().Time.Unix()
	err = kp.CloseSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}
	tags := fromTags.AppendTags(toTags)
	return sdk.Result{Tags: tags}
}
