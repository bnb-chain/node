package bridge

import (
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/bridge/types"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case TransferInMsg:
			return handleTransferInMsg(ctx, keeper, msg)
		case TransferOutMsg:
			return handleTransferOutMsg(ctx, keeper, msg)
		case BindMsg:
			return handleBindMsg(ctx, keeper, msg)
		case TimeoutMsg:
			return handleTimeoutMsg(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized bridge msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleTransferInMsg(ctx sdk.Context, bridgeKeeper Keeper, msg TransferInMsg) sdk.Result {
	currentSequence := bridgeKeeper.GetCurrentSequence(ctx, types.KeyCurrentTransferSequence)
	if msg.Sequence != currentSequence {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence is %d", currentSequence)).Result()
	}

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
	currentSequence := bridgeKeeper.GetCurrentSequence(ctx, types.KeyTimeoutSequence)
	if msg.Sequence != currentSequence {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence is %d", currentSequence)).Result()
	}

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

func handleBindMsg(ctx sdk.Context, keeper Keeper, msg BindMsg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)

	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can bind token %s", msg.Symbol)).Result()
	}

	err = keeper.TokenMapper.UpdateBind(ctx, symbol, msg.ContractAddress.String(), msg.ContractDecimal)
	if err != nil {
		return sdk.ErrInternal(fmt.Sprintf("update token bind info error")).Result()
	}

	return sdk.Result{}
}

func handleTransferOutMsg(ctx sdk.Context, keeper Keeper, msg TransferOutMsg) sdk.Result {
	if !time.Unix(msg.ExpireTime, 0).After(ctx.BlockHeader().Time.Add(types.MinTransferOutExpireTimeGap)) {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time should be %d seconds after now(%s)",
			types.MinTransferOutExpireTimeGap, ctx.BlockHeader().Time.UTC().String())).Result()
	}

	symbol := strings.ToUpper(msg.Amount.Denom)

	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", symbol)).Result()
	}

	if token.ContractAddress == "" {
		return types.ErrTokenNotBound(fmt.Sprintf("token %s is not bound", symbol)).Result()
	}

	_, cErr := keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, sdk.Coins{msg.Amount})
	if cErr != nil {
		return cErr.Result()
	}

	return sdk.Result{}
}
