package burn

import (
	"fmt"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/log"
	common "github.com/binance-chain/node/common/types"
	miniToken "github.com/binance-chain/node/plugins/minitokens"
	"github.com/binance-chain/node/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, miniTokenMapper miniToken.MiniTokenMapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(BurnMsg); ok {
			symbol := strings.ToUpper(msg.Symbol)
			if common.IsMiniTokenSymbol(symbol) {
				return handleBurnMiniToken(ctx, miniTokenMapper, keeper, msg)
			} else {
				return handleBurnToken(ctx, tokenMapper, keeper, msg)
			}
		}
		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleBurnToken(ctx sdk.Context, tokenMapper store.Mapper, keeper bank.Keeper, msg BurnMsg) sdk.Result {
	logger := log.With("module", "token", "symbol", msg.Symbol, "amount", msg.Amount)
	burnAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	token, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		logger.Info("burn token failed", "reason", "invalid token symbol")
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !token.IsOwner(msg.From) {
		logger.Info("burn token failed", "reason", "not token's owner", "from", msg.From, "owner", token.Owner)
		return sdk.ErrUnauthorized("only the owner of the token can burn the token").Result()
	}

	coins := keeper.GetCoins(ctx, token.Owner)
	if coins.AmountOf(symbol) < burnAmount ||
		token.TotalSupply.ToInt64() < burnAmount {
		logger.Info("burn token failed", "reason", "no enough tokens to burn")
		return sdk.ErrInsufficientCoins("do not have enough token to burn").Result()
	}

	_, _, sdkError := keeper.SubtractCoins(ctx, token.Owner, sdk.Coins{{
		Denom:  symbol,
		Amount: burnAmount,
	}})
	if sdkError != nil {
		logger.Error("burn token failed", "reason", "subtract tokens failed: "+sdkError.Error())
		return sdkError.Result()
	}

	newTotalSupply := token.TotalSupply.ToInt64() - burnAmount
	err = tokenMapper.UpdateTotalSupply(ctx, symbol, newTotalSupply)
	if err != nil {
		logger.Error("burn token failed", "reason", "update total supply failed: "+err.Error())
		return sdk.ErrInternal(err.Error()).Result()
	}

	logger.Info("successfully burnt token", "NewTotalSupply", newTotalSupply)
	return sdk.Result{}
}

func handleBurnMiniToken(ctx sdk.Context, tokenMapper miniToken.MiniTokenMapper, keeper bank.Keeper, msg BurnMsg) sdk.Result {
	logger := log.With("module", "miniToken", "symbol", msg.Symbol, "amount", msg.Amount)
	burnAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	token, err := tokenMapper.GetToken(ctx, symbol)
	errLogMsg := "burn token failed"
	if err != nil {
		logger.Info("burn token failed", "reason", "invalid token symbol")
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !token.IsOwner(msg.From) {
		logger.Info("burn token failed", "reason", "not token's owner", "from", msg.From, "owner", token.Owner)
		return sdk.ErrUnauthorized("only the owner of the token can burn the token").Result()
	}

	coins := keeper.GetCoins(ctx, token.Owner)

	useAllBalance := coins.AmountOf(symbol) == burnAmount

	if burnAmount <= 0 || (!useAllBalance && (burnAmount < common.MiniTokenMinTotalSupply)) {
		logger.Info(errLogMsg, "reason", "unfreeze amount doesn't reach the min supply")
		return sdk.ErrInvalidCoins(fmt.Sprintf("freeze amount is too small, the min amount is %d or total frozen balance",
			common.MiniTokenMinTotalSupply)).Result()
	}

	if coins.AmountOf(symbol) < burnAmount ||
		token.TotalSupply.ToInt64() < burnAmount {
		logger.Info("burn token failed", "reason", "no enough tokens to burn")
		return sdk.ErrInsufficientCoins("do not have enough token to burn").Result()
	}

	_, _, sdkError := keeper.SubtractCoins(ctx, token.Owner, sdk.Coins{{
		Denom:  symbol,
		Amount: burnAmount,
	}})
	if sdkError != nil {
		logger.Error("burn token failed", "reason", "subtract tokens failed: "+sdkError.Error())
		return sdkError.Result()
	}

	newTotalSupply := token.TotalSupply.ToInt64() - burnAmount
	err = tokenMapper.UpdateTotalSupply(ctx, symbol, newTotalSupply)
	if err != nil {
		logger.Error("burn token failed", "reason", "update total supply failed: "+err.Error())
		return sdk.ErrInternal(err.Error()).Result()
	}

	logger.Info("successfully burnt token", "NewTotalSupply", newTotalSupply)
	return sdk.Result{}
}
