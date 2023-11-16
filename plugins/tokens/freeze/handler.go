package freeze

import (
	"fmt"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/bnb-chain/node/common/log"
	common "github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/plugins/tokens/store"
)

// NewHandler creates a new token freeze message handler
func NewHandler(tokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case FreezeMsg:
			if sdk.IsUpgrade(sdk.BCFusionFirstHardFork) {
				return sdk.ErrMsgNotSupported("").Result()
			}
			return handleFreezeToken(ctx, tokenMapper, accKeeper, keeper, msg)
		case UnfreezeMsg:
			return handleUnfreezeToken(ctx, tokenMapper, accKeeper, keeper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleFreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper, msg FreezeMsg) sdk.Result {
	if sdk.IsUpgrade(sdk.BCFusionFirstHardFork) {
		return sdk.ErrMsgNotSupported("").Result()
	}
	freezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", freezeAmount, "addr", msg.From)
	coins := keeper.GetCoins(ctx, msg.From)
	balance := coins.AmountOf(symbol)
	if balance < freezeAmount {
		logger.Info("freeze token failed", "reason", "no enough free tokens to freeze")
		return sdk.ErrInsufficientCoins("do not have enough token to freeze").Result()
	}

	if common.IsMiniTokenSymbol(symbol) {
		if msg.Amount < common.MiniTokenMinExecutionAmount && balance != freezeAmount {
			logger.Info("freeze token failed", "reason", "freeze amount doesn't reach the min amount")
			return sdk.ErrInvalidCoins(fmt.Sprintf("freeze amount is too small, the min amount is %d or total account balance",
				common.MiniTokenMinExecutionAmount)).Result()
		}
	}

	account := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	newFrozenTokens := account.GetFrozenCoins().Plus(sdk.Coins{{Denom: symbol, Amount: freezeAmount}})
	newFreeTokens := account.GetCoins().Minus(sdk.Coins{{Denom: symbol, Amount: freezeAmount}})
	account.SetFrozenCoins(newFrozenTokens)
	_ = account.SetCoins(newFreeTokens)
	accKeeper.SetAccount(ctx, account)
	logger.Info("finish freezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}

func handleUnfreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper, msg UnfreezeMsg) sdk.Result {
	unfreezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", unfreezeAmount, "addr", msg.From)

	account := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	frozenAmount := account.GetFrozenCoins().AmountOf(symbol)
	if frozenAmount < unfreezeAmount {
		logger.Info("unfreeze token failed", "reason", "no enough frozen tokens to unfreeze")
		return sdk.ErrInsufficientCoins("do not have enough token to unfreeze").Result()
	}

	if common.IsMiniTokenSymbol(symbol) {
		if unfreezeAmount < common.MiniTokenMinExecutionAmount && frozenAmount != unfreezeAmount {
			logger.Info("unfreeze token failed", "reason", "unfreeze amount doesn't reach the min amount")
			return sdk.ErrInvalidCoins(fmt.Sprintf("unfreeze amount is too small, the min amount is %d or total frozen balance",
				common.MiniTokenMinExecutionAmount)).Result()
		}
	}

	newFrozenTokens := account.GetFrozenCoins().Minus(sdk.Coins{{Denom: symbol, Amount: unfreezeAmount}})
	newFreeTokens := account.GetCoins().Plus(sdk.Coins{{Denom: symbol, Amount: unfreezeAmount}})
	account.SetFrozenCoins(newFrozenTokens)
	_ = account.SetCoins(newFreeTokens)
	accKeeper.SetAccount(ctx, account)
	logger.Debug("finish unfreezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}
