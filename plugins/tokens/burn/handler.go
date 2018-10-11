package burn

import (
	"github.com/BiJie/BinanceChain/common/log"
	"reflect"
	"strings"

	common "github.com/BiJie/BinanceChain/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, keeper bank.Keeper) common.Handler {
	return func(ctx sdk.Context, msg sdk.Msg, simulate bool) sdk.Result {
		if msg, ok := msg.(Msg); ok {
			return handleBurnToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleBurnToken(ctx sdk.Context, tokenMapper store.Mapper, keeper bank.Keeper, msg Msg) sdk.Result {
	logger := log.With("module", "token")
	logger.Info("start burning token", "symbol", msg.Symbol, "amount", msg.Amount)
	burnAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	token, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		logger.Info("burn token failed", "symbol", symbol, "reason", "invalid token symbol")
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !token.IsOwner(msg.From) {
		logger.Info("burn token failed", "symbol", symbol, "reason", "not token's owner")
		return sdk.ErrUnauthorized("only the owner of the token can burn the token").Result()
	}

	coins := keeper.GetCoins(ctx, token.Owner)
	if coins.AmountOf(symbol).Int64() < burnAmount ||
		token.TotalSupply.ToInt64() < burnAmount {
		logger.Info("burn token failed", "symbol", symbol, "reason", "no enough tokens to burn")
		return sdk.ErrInsufficientCoins("do not have enough token to burn").Result()
	}

	logger.Info("subtract tokens from the owner's balance", "symbol", symbol, "owner", token.Owner, "amount", burnAmount)
	_, _, sdkError := keeper.SubtractCoins(ctx, token.Owner,
		append((sdk.Coins)(nil), sdk.Coin{
			Denom:  symbol,
			Amount: sdk.NewInt(burnAmount),
		}))
	if sdkError != nil {
		logger.Info("burn token failed","symbol", symbol, "reason", "subtract tokens failed: " + sdkError.Error())
		return sdkError.Result()
	}

	newTotalSupply := token.TotalSupply.ToInt64()-burnAmount
	logger.Info("update token's total supply", "symbol", symbol, "old", token.TotalSupply.ToInt64(), "new", newTotalSupply)
	err = tokenMapper.UpdateTotalSupply(ctx, symbol, newTotalSupply)
	if err != nil {
		logger.Info("burn token failed", "symbol", symbol, "reason", "update total supply failed: " + err.Error())
		return sdk.ErrInternal(err.Error()).Result()
	}

	logger.Info("successfully burnt token", "symbol", msg.Symbol, "amount", msg.Amount)
	return sdk.Result{}
}
