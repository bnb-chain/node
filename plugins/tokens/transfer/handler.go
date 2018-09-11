package transfer

import (
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/account"
	"github.com/BiJie/BinanceChain/common/types"
)

// NewHandler returns a handler for "bank" type messages.
func NewHandler(k account.Keeper) types.Handler {
	return func(ctx types.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case bank.MsgSend:
			return handleTransfer(ctx, k, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

// Handle MsgTransfer.
func handleTransfer(ctx types.Context, k account.Keeper, msg bank.MsgSend) sdk.Result {
	// NOTE: totalIn == totalOut should already have been checked
	tags, err := k.InputOutputCoins(ctx, msg.Inputs, msg.Outputs)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{
		Tags: tags,
	}
}
