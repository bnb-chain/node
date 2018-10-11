package freeze

import (
	"github.com/BiJie/BinanceChain/common/log"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case FreezeMsg:
			return handleFreezeToken(ctx, tokenMapper, accountMapper, keeper, msg)
		case UnfreezeMsg:
			return handleUnfreezeToken(ctx, tokenMapper, accountMapper, keeper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleFreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper, msg FreezeMsg) sdk.Result {
	freezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", freezeAmount, "addr", msg.From)
	logger.Info("start freezing token")
	coins := keeper.GetCoins(ctx, msg.From)
	if coins.AmountOf(symbol).Int64() < freezeAmount {
		logger.Info("freeze token failed", "reason", "no enough free tokens to freeze")
		return sdk.ErrInsufficientCoins("do not have enough token to freeze").Result()
	}

	logger.Info("subtract token from free balance")
	_, _, sdkError := keeper.SubtractCoins(ctx, msg.From, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: sdk.NewInt(freezeAmount)}))
	if sdkError != nil {
		// should not happen because we have checked balance >= freezeAmount
		logger.Error("freeze token failed", "reason", "subtract token failed:" + sdkError.Error())
		return sdkError.Result()
	}

	logger.Info("update frozen balance")
	updateFrozenOfAccount(ctx, accountMapper, msg.From, symbol, freezeAmount)
	logger.Info("finish freezing token")
	return sdk.Result{}
}

func handleUnfreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper, msg UnfreezeMsg) sdk.Result {
	unfreezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", unfreezeAmount, "addr", msg.From)
	logger.Info("start unfreezing token")
	account := accountMapper.GetAccount(ctx, msg.From).(types.NamedAccount)
	frozenAmount := account.GetFrozenCoins().AmountOf(symbol).Int64()
	if frozenAmount < unfreezeAmount {
		logger.Info("unfreeze token failed", "reason", "no enough frozen tokens to unfreeze")
		return sdk.ErrInsufficientCoins("do not have enough token to unfreeze").Result()
	}

	logger.Info("update frozen balance")
	newFrozenTokens := account.GetFrozenCoins().Minus(append(sdk.Coins{}, sdk.Coin{Denom: symbol, Amount: sdk.NewInt(unfreezeAmount)}))
	account.SetFrozenCoins(newFrozenTokens)
	accountMapper.SetAccount(ctx, account)
	logger.Debug("updated frozen balance", "newVal", newFrozenTokens)

	logger.Info("add tokens to free balance")
	_, _, sdkError := keeper.AddCoins(ctx, msg.From, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: sdk.NewInt(unfreezeAmount)}))
	if sdkError != nil {
		// should not happen because we have checked the unfreezeAmount > 0
		logger.Error("unfreeze token failed", "reason", "add token failed: " + sdkError.Error())
		return sdkError.Result()
	}

	logger.Info("finish unfreezing token")
	return sdk.Result{}
}

func updateFrozenOfAccount(ctx sdk.Context, accountMapper auth.AccountMapper, address sdk.AccAddress, symbol string, frozenAmount int64) {
	account := accountMapper.GetAccount(ctx, address).(types.NamedAccount)
	account.SetFrozenCoins(account.GetFrozenCoins().Plus(append(sdk.Coins{}, sdk.Coin{Denom: symbol, Amount: sdk.NewInt(frozenAmount)})))
	accountMapper.SetAccount(ctx, account)
}
