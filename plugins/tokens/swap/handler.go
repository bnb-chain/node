package swap

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewHandler(kp Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case HTLTMsg:
			return handleHashTimerLockedTransfer(ctx, kp, msg)
		case DepositHTLTMsg:
			return handleDepositHashTimerLockedTransfer(ctx, kp, msg)
		case ClaimHTLTMsg:
			return handleClaimHashTimerLockedTransfer(ctx, kp, msg)
		case RefundHTLTMsg:
			return handleRefundHashTimerLockedTransfer(ctx, kp, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleHashTimerLockedTransfer(ctx sdk.Context, kp Keeper, msg HTLTMsg) sdk.Result {
	blockTime := ctx.BlockHeader().Time.Unix()
	if msg.Timestamp < blockTime-TwoHours || msg.Timestamp > blockTime+OneHour {
		return ErrInvalidTimestamp(fmt.Sprintf("The timestamp (%d) should not be one hour ahead or two hours behind block time (%d)", msg.Timestamp, ctx.BlockHeader().Time.Unix())).Result()
	}
	swap := &AtomicSwap{
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
	err := kp.CreateSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}
	tags, err := kp.ck.SendCoins(ctx, msg.From, AtomicSwapCoinsAccAddr, sdk.Coins{msg.OutAmount})
	if err != nil {
		return err.Result()
	}
	return sdk.Result{Tags: tags}
}

func handleDepositHashTimerLockedTransfer(ctx sdk.Context, kp Keeper, msg DepositHTLTMsg) sdk.Result {
	swap := kp.GetSwap(ctx, msg.RandomNumberHash)
	if swap == nil {
		return ErrNonExistRandomNumberHash(fmt.Sprintf("No matched swap with randomNumberHash %v", msg.RandomNumberHash)).Result()
	}
	if swap.CrossChain {
		return ErrInvalidSingleChainSwap("Can't deposit to cross chain swap").Result()
	}
	if swap.Status != Open {
		return ErrInvalidSingleChainSwap(fmt.Sprintf("Expected swap status is open, acutually it is %s", swap.Status.String())).Result()
	}
	if ctx.BlockHeight() >= swap.ExpireHeight {
		return ErrInvalidSingleChainSwap(fmt.Sprintf("Current block height is %d, the swap expire height(%d) is passed", ctx.BlockHeight(), swap.ExpireHeight)).Result()
	}
	if !bytes.Equal(swap.From, msg.To) || !bytes.Equal(swap.To, msg.From) {
		return ErrInvalidSingleChainSwap(fmt.Sprintf("Addresses don't match, expected deposit from %s and recipient %s", swap.To.String(), swap.From.String())).Result()
	}
	if !swap.InAmount.IsZero() {
		return ErrInvalidSingleChainSwap("Can't deposit a swap for multiple times").Result()
	}
	swap.InAmount = msg.OutAmount
	err := kp.UpdateSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}

	tags, err := kp.ck.SendCoins(ctx, msg.From, AtomicSwapCoinsAccAddr, sdk.Coins{msg.OutAmount})
	if err != nil {
		return err.Result()
	}

	return sdk.Result{Tags: tags}

}

func handleClaimHashTimerLockedTransfer(ctx sdk.Context, kp Keeper, msg ClaimHTLTMsg) sdk.Result {
	swap := kp.GetSwap(ctx, msg.RandomNumberHash)
	if swap == nil {
		return ErrNonExistRandomNumberHash(fmt.Sprintf("No matched swap with randomNumberHash %v", msg.RandomNumberHash)).Result()
	}
	if swap.Status != Open {
		return ErrUnexpectedSwapStatus(fmt.Sprintf("Expected swap status is Open, actually it is %s", swap.Status.String())).Result()
	}
	if swap.ExpireHeight <= ctx.BlockHeight() {
		return ErrClaimExpiredSwap(fmt.Sprintf("Current block height is %d, the swap expire height(%d) is passed", ctx.BlockHeight(), swap.ExpireHeight)).Result()
	}

	if !bytes.Equal(CalculateRandomHash(msg.RandomNumber, swap.Timestamp), msg.RandomNumberHash) {
		return ErrMismatchedRandomNumber(fmt.Sprintf("Mismatched random number")).Result()
	}

	if !swap.CrossChain && swap.InAmount.IsZero() {
		return ErrUnexpectedClaimSingleChainSwap("Can't claim a single chain swap which has not been deposited").Result()
	}

	tags := sdk.EmptyTags()
	if !swap.OutAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.To, sdk.Coins{swap.OutAmount})
		if err != nil {
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
	}
	if !swap.InAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.From, sdk.Coins{swap.InAmount})
		if err != nil {
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
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
	err := kp.CloseSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}
	return sdk.Result{Tags: tags}
}

func handleRefundHashTimerLockedTransfer(ctx sdk.Context, kp Keeper, msg RefundHTLTMsg) sdk.Result {
	swap := kp.GetSwap(ctx, msg.RandomNumberHash)
	if swap == nil {
		return ErrNonExistRandomNumberHash(fmt.Sprintf("No matched swap with randomNumberHash %v", msg.RandomNumberHash)).Result()
	}
	if swap.Status != Open {
		return ErrUnexpectedSwapStatus(fmt.Sprintf("Expected swap status is Open, actually it is %s", swap.Status.String())).Result()
	}
	if ctx.BlockHeight() < swap.ExpireHeight {
		return ErrRefundUnexpiredSwap(fmt.Sprintf("Current block height is %d, the expire height (%d) is still not reached", ctx.BlockHeight(), swap.ExpireHeight)).Result()
	}

	tags := sdk.EmptyTags()
	if !swap.OutAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.From, sdk.Coins{swap.OutAmount})
		if err != nil {
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
	}
	if !swap.InAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.To, sdk.Coins{swap.InAmount})
		if err != nil {
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
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
	err := kp.CloseSwap(ctx, swap)
	if err != nil {
		return err.Result()
	}
	return sdk.Result{Tags: tags}
}
