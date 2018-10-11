package issue

import (
	"fmt"
	"github.com/BiJie/BinanceChain/common/log"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(Msg); ok {
			return handleIssueToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleIssueToken(ctx sdk.Context, tokenMapper store.Mapper, keeper bank.Keeper, msg Msg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)
	exists := tokenMapper.Exists(ctx, symbol)
	logger := log.With("module", "token", "symbol", symbol)
	logger.Info("start issuing token", "name", msg.Name, "total_supply", msg.TotalSupply, "issuer", msg.From)
	if exists {
		logger.Info("issue token failed", "reason", "already exists")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	logger.Info("add to token store")
	token := types.NewToken(msg.Name, symbol, msg.TotalSupply, msg.From)
	err := tokenMapper.NewToken(ctx, token)
	if err != nil {
		logger.Error("issue token failed", "reason", "add token failed: " + err.Error())
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	logger.Info("update owner's balance")
	_, _, sdkError := keeper.AddCoins(ctx, token.Owner,
		append((sdk.Coins)(nil), sdk.Coin{
			Denom:  token.Symbol,
			Amount: sdk.NewInt(token.TotalSupply.ToInt64()),
		}))
	if sdkError != nil {
		logger.Info("issue token failed", "reason", "update balance failed: " + sdkError.Error())
		return sdkError.Result()
	}

	logger.Info("finish issuing token")
	return sdk.Result{}
}
