package freeze

import (
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/log"
	common "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

// NewHandler creates a new token freeze message handler
func NewHandler(tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) common.Handler {
	return func(ctx sdk.Context, msg sdk.Msg, simulate bool) sdk.Result {
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
	coins := keeper.GetCoins(ctx, msg.From)
	if coins.AmountOf(symbol).Int64() < freezeAmount {
		logger.Info("freeze token failed", "reason", "no enough free tokens to freeze")
		return sdk.ErrInsufficientCoins("do not have enough token to freeze").Result()
	}

	account := accountMapper.GetAccount(ctx, msg.From).(common.NamedAccount)
	newFrozenTokens := account.GetFrozenCoins().Plus(sdk.Coins{{Denom: symbol, Amount: sdk.NewInt(freezeAmount)}})
	newFreeTokens := account.GetCoins().Minus(sdk.Coins{{Denom: symbol, Amount: sdk.NewInt(freezeAmount)}})
	account.SetFrozenCoins(newFrozenTokens)
	account.SetCoins(newFreeTokens)
	accountMapper.SetAccount(ctx, account)
	logger.Info("finish freezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}

func handleUnfreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper, msg UnfreezeMsg) sdk.Result {
	unfreezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", unfreezeAmount, "addr", msg.From)
	account := accountMapper.GetAccount(ctx, msg.From).(common.NamedAccount)
	frozenAmount := account.GetFrozenCoins().AmountOf(symbol).Int64()
	if frozenAmount < unfreezeAmount {
		logger.Info("unfreeze token failed", "reason", "no enough frozen tokens to unfreeze")
		return sdk.ErrInsufficientCoins("do not have enough token to unfreeze").Result()
	}

	newFrozenTokens := account.GetFrozenCoins().Minus(sdk.Coins{{Denom: symbol, Amount: sdk.NewInt(unfreezeAmount)}})
	newFreeTokens := account.GetCoins().Plus(sdk.Coins{{Denom: symbol, Amount: sdk.NewInt(unfreezeAmount)}})
	account.SetFrozenCoins(newFrozenTokens)
	account.SetCoins(newFreeTokens)
	accountMapper.SetAccount(ctx, account)
	logger.Debug("finish unfreezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}
