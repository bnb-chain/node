package bridge

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/bridge/types"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case TransferMsg:
			return handleTransferMsg(ctx, keeper, msg)
		case TimeoutMsg:
			return handleTimeoutMsg(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized bridge msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleTransferMsg(ctx sdk.Context, bridgeKeeper Keeper, msg TransferMsg) sdk.Result {
	// TODO reject wrong sequence
	claim, err := types.CreateOracleClaimFromTransferMsg(msg)
	if err != nil {
		return err.Result()
	}

	_, err = bridgeKeeper.ProcessTransferClaim(ctx, claim)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

func handleTimeoutMsg(ctx sdk.Context, bridgeKeeper Keeper, msg TimeoutMsg) sdk.Result {
	// TODO reject wrong sequence
	claim, err := types.CreateOracleClaimFromTimeoutMsg(msg)
	if err != nil {
		return err.Result()
	}

	_, err = bridgeKeeper.ProcessTransferClaim(ctx, claim)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{}
}
