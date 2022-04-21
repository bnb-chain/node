package bank

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewHandler returns a handler for "bank" type messages.
func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case MsgSend:
			return handleMsgSend(ctx, k, msg)
		default:
			errMsg := "Unrecognized bank Msg type: %s" + msg.Type()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

// Handle MsgSend.
func handleMsgSend(ctx sdk.Context, k Keeper, msg MsgSend) sdk.Result {
	logger := ctx.Logger()
	for _, script := range sdk.GetRegisteredScripts(msg.Type()) {
		if script == nil {
			logger.Error(fmt.Sprintf("Empty script is specified for msg %s", msg.Type()))
			continue
		}
		if err := script(ctx, msg); err != nil {
			return err.Result()
		}
	}

	if sdk.IsUpgrade(sdk.BEP8) {
		am := k.GetAccountKeeper()
		for _, in := range msg.Inputs {
			if err := CheckAndValidateMiniTokenCoins(ctx, am, in.Address, in.Coins); err != nil {
				return err.Result()
			}
		}
	}
	// NOTE: totalIn == totalOut should already have been checked
	tags, err := k.InputOutputCoins(ctx, msg.Inputs, msg.Outputs)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{
		Tags: tags,
	}
}
