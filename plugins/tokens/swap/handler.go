package swap

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/binance-chain/node/common/types"

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
	symbolError := types.ValidateMapperTokenCoins(msg.Amount)
	if symbolError != nil {
		return sdk.ErrInvalidCoins(symbolError.Error()).Result()
	}
	blockTime := ctx.BlockHeader().Time.Unix()
	if msg.Timestamp < blockTime-ThirtyMinutes || msg.Timestamp > blockTime+FifteenMinutes {
		return ErrInvalidTimestamp(fmt.Sprintf("Timestamp (%d) can neither be 15 minutes ahead of the current time (%d), nor 30 minutes later", msg.Timestamp, ctx.BlockHeader().Time.Unix())).Result()
	}
	tags, err := kp.ck.SendCoins(ctx, msg.From, AtomicSwapCoinsAccAddr, msg.Amount)
	if err != nil {
		return err.Result()
	}
	swap := &AtomicSwap{
		From:                msg.From,
		To:                  msg.To,
		OutAmount:           msg.Amount,
		InAmount:            nil,
		ExpectedIncome:      msg.ExpectedIncome,
		RecipientOtherChain: msg.RecipientOtherChain,
		RandomNumberHash:    msg.RandomNumberHash,
		RandomNumber:        nil,
		Timestamp:           msg.Timestamp,
		ExpireHeight:        ctx.BlockHeight() + int64(msg.HeightSpan),
		CrossChain:          msg.CrossChain,
		ClosedTime:          0,
		Status:              Open,
		Index:               kp.getIndex(ctx),
	}
	swapID := CalculateSwapID(swap.RandomNumberHash, swap.From, msg.SenderOtherChain)
	err = kp.CreateSwap(ctx, swapID, swap)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{Tags: tags, Data: swapID, Log: fmt.Sprintf("swapID: %s", hex.EncodeToString(swapID))}
}

func handleDepositHashTimerLockedTransfer(ctx sdk.Context, kp Keeper, msg DepositHTLTMsg) sdk.Result {
	symbolError := types.ValidateMapperTokenCoins(msg.Amount)
	if symbolError != nil {
		return sdk.ErrInvalidCoins(symbolError.Error()).Result()
	}
	swap := kp.GetSwap(ctx, msg.SwapID)
	if swap == nil {
		return ErrNonExistSwapID(fmt.Sprintf("No matched swap with swapID %v", msg.SwapID)).Result()
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
	if !bytes.Equal(swap.To, msg.From) {
		return ErrInvalidSingleChainSwap(fmt.Sprintf("Addresses don't match, expected deposit from %s and recipient %s", swap.To.String(), swap.From.String())).Result()
	}
	if !swap.InAmount.IsZero() {
		return ErrInvalidSingleChainSwap("Can't deposit a swap for multiple times").Result()
	}
	tags, err := kp.ck.SendCoins(ctx, msg.From, AtomicSwapCoinsAccAddr, msg.Amount)
	if err != nil {
		return err.Result()
	}

	swap.InAmount = msg.Amount
	err = kp.UpdateSwap(ctx, msg.SwapID, swap)
	if err != nil {
		kp.logger.Error("Failed to update swap", "err", err.Error())
		return err.Result()
	}

	return sdk.Result{Tags: tags}

}

func handleClaimHashTimerLockedTransfer(ctx sdk.Context, kp Keeper, msg ClaimHTLTMsg) sdk.Result {
	swap := kp.GetSwap(ctx, msg.SwapID)
	if swap == nil {
		return ErrNonExistSwapID(fmt.Sprintf("No matched swap with swapID %v", msg.SwapID)).Result()
	}
	if swap.Status != Open {
		return ErrUnexpectedSwapStatus(fmt.Sprintf("Expected swap status is Open, actually it is %s", swap.Status.String())).Result()
	}
	if swap.ExpireHeight <= ctx.BlockHeight() {
		return ErrClaimExpiredSwap(fmt.Sprintf("Current block height is %d, the swap expire height(%d) is passed", ctx.BlockHeight(), swap.ExpireHeight)).Result()
	}

	if !bytes.Equal(CalculateRandomHash(msg.RandomNumber, swap.Timestamp), swap.RandomNumberHash) {
		return ErrMismatchedRandomNumber(fmt.Sprintf("Mismatched random number")).Result()
	}

	if !swap.CrossChain && swap.InAmount.IsZero() {
		return ErrUnexpectedClaimSingleChainSwap("Can't claim a single chain swap which has not been deposited").Result()
	}

	tags := sdk.EmptyTags()
	if !swap.OutAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.To, swap.OutAmount)
		if err != nil {
			kp.logger.Error("Failed to send coins", "sender", AtomicSwapCoinsAccAddr.String(), "recipient", swap.To.String(), "amount", swap.OutAmount.String(), "err", err.Error())
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
	}
	if !swap.InAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.From, swap.InAmount)
		if err != nil {
			kp.logger.Error("Failed to send coins", "sender", AtomicSwapCoinsAccAddr.String(), "recipient", swap.From.String(), "amount", swap.InAmount.String(), "err", err.Error())
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
	}
	if ctx.IsDeliverTx() && kp.addrPool != nil {
		if !bytes.Equal(msg.From, swap.From) && !swap.InAmount.IsZero() {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.From})
		}
		if !bytes.Equal(msg.From, swap.To) && !swap.OutAmount.IsZero() {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.To})
		}
	}

	swap.RandomNumber = msg.RandomNumber
	swap.Status = Completed
	swap.ClosedTime = ctx.BlockHeader().Time.Unix()
	err := kp.CloseSwap(ctx, msg.SwapID, swap)
	if err != nil {
		kp.logger.Error("Failed to close swap", "err", err.Error())
		return err.Result()
	}
	return sdk.Result{Tags: tags}
}

func handleRefundHashTimerLockedTransfer(ctx sdk.Context, kp Keeper, msg RefundHTLTMsg) sdk.Result {
	swap := kp.GetSwap(ctx, msg.SwapID)
	if swap == nil {
		return ErrNonExistSwapID(fmt.Sprintf("No matched swap with swapID %v", msg.SwapID)).Result()
	}
	if swap.Status != Open {
		return ErrUnexpectedSwapStatus(fmt.Sprintf("Expected swap status is Open, actually it is %s", swap.Status.String())).Result()
	}
	if ctx.BlockHeight() < swap.ExpireHeight {
		return ErrRefundUnexpiredSwap(fmt.Sprintf("Current block height is %d, the expire height (%d) is still not reached", ctx.BlockHeight(), swap.ExpireHeight)).Result()
	}

	tags := sdk.EmptyTags()
	if !swap.OutAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.From, swap.OutAmount)
		if err != nil {
			kp.logger.Error("Failed to send coins", "sender", AtomicSwapCoinsAccAddr.String(), "recipient", swap.From.String(), "amount", swap.OutAmount.String(), "err", err.Error())
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
	}
	if !swap.InAmount.IsZero() {
		sendCoinTags, err := kp.ck.SendCoins(ctx, AtomicSwapCoinsAccAddr, swap.To, swap.InAmount)
		if err != nil {
			kp.logger.Error("Failed to send coins", "sender", AtomicSwapCoinsAccAddr.String(), "recipient", swap.To.String(), "amount", swap.InAmount.String(), "err", err.Error())
			return err.Result()
		}
		tags = tags.AppendTags(sendCoinTags)
	}
	if ctx.IsDeliverTx() && kp.addrPool != nil {
		if !bytes.Equal(msg.From, swap.From) && !swap.OutAmount.IsZero() {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.From})
		}
		if !bytes.Equal(msg.From, swap.To) && !swap.InAmount.IsZero() {
			kp.addrPool.AddAddrs([]sdk.AccAddress{swap.To})
		}
	}

	swap.Status = Expired
	swap.ClosedTime = ctx.BlockHeader().Time.Unix()
	err := kp.CloseSwap(ctx, msg.SwapID, swap)
	if err != nil {
		kp.logger.Error("Failed to close swap", "err", err.Error())
		return err.Result()
	}
	return sdk.Result{Tags: tags}
}
