package slashing

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		// NOTE msg already has validate basic run
		switch msg := msg.(type) {
		case MsgSideChainUnjail:
			return handleMsgSideChainUnjail(ctx, msg, k)
		case MsgBscSubmitEvidence:
			return handleMsgBscSubmitEvidence(ctx, msg, k)
		default:
			return sdk.ErrTxDecode("invalid message parse in staking module").Result()
		}
	}
}

func NewSlashingHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		// NOTE msg already has validate basic run
		switch msg := msg.(type) {
		case MsgUnjail:
			return handleMsgUnjail(ctx, msg, k)
		default:
			return sdk.ErrTxDecode("invalid message parse in staking module").Result()
		}
	}
}

// Validators must submit a transaction to unjail itself after
// having been jailed (and thus unbonded) for downtime
func handleMsgUnjail(ctx sdk.Context, msg MsgUnjail, k Keeper) sdk.Result {
	if err := k.Unjail(ctx, msg.ValidatorAddr); err != nil {
		return err.Result()
	}

	tags := sdk.NewTags("action", []byte("unjail"), "validator", []byte(msg.ValidatorAddr.String()))

	return sdk.Result{
		Tags: tags,
	}
}
