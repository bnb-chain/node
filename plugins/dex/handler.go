package dex

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewHandler - returns a handler for dex type messages.
func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case MakeOfferMsg:
			return handleMakeOffer(ctx, k, msg)
		case FillOfferMsg:
			return handleFillOffer(ctx, k, msg)
		case CancelOfferMsg:
			return handleCancelOffer(ctx, k, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

// Handle MakeOffer -
func handleMakeOffer(ctx sdk.Context, k Keeper, msg MakeOfferMsg) sdk.Result {
	return sdk.Result{}
}

// Handle FillOffer -
func handleFillOffer(ctx sdk.Context, k Keeper, msg FillOfferMsg) sdk.Result {
	return sdk.Result{}
}

// Handle CancelOffer -
func handleCancelOffer(ctx sdk.Context, k Keeper, msg CancelOfferMsg) sdk.Result {
	return sdk.Result{}
}
