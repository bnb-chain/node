package freeze

import (
	"fmt"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/log"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/store"
)

// NewHandler creates a new token freeze message handler
func NewHandler(tokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case FreezeMsg:
			symbol := strings.ToUpper(msg.Symbol)
			if common.IsMiniTokenSymbol(symbol) {
				return handleFreezeMiniToken(ctx, tokenMapper, accKeeper, keeper, msg)
			} else {
				return handleFreezeToken(ctx, tokenMapper, accKeeper, keeper, msg)
			}
		case UnfreezeMsg:
			symbol := strings.ToUpper(msg.Symbol)
			if common.IsMiniTokenSymbol(symbol) {
				return handleUnfreezeMiniToken(ctx, tokenMapper, accKeeper, keeper, msg)
			} else {
				return handleUnfreezeToken(ctx, tokenMapper, accKeeper, keeper, msg)
			}
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleFreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper, msg FreezeMsg) sdk.Result {
	freezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", freezeAmount, "addr", msg.From)
	coins := keeper.GetCoins(ctx, msg.From)
	if coins.AmountOf(symbol) < freezeAmount {
		logger.Info("freeze token failed", "reason", "no enough free tokens to freeze")
		return sdk.ErrInsufficientCoins("do not have enough token to freeze").Result()
	}

	account := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	newFrozenTokens := account.GetFrozenCoins().Plus(sdk.Coins{{Denom: symbol, Amount: freezeAmount}})
	newFreeTokens := account.GetCoins().Minus(sdk.Coins{{Denom: symbol, Amount: freezeAmount}})
	account.SetFrozenCoins(newFrozenTokens)
	account.SetCoins(newFreeTokens)
	accKeeper.SetAccount(ctx, account)
	logger.Info("finish freezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}

func handleUnfreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper, msg UnfreezeMsg) sdk.Result {
	unfreezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", unfreezeAmount, "addr", msg.From)

	_, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		logger.Info("unfreeze token failed", "reason", "symbol not exist")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}
	account := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	frozenAmount := account.GetFrozenCoins().AmountOf(symbol)
	if frozenAmount < unfreezeAmount {
		logger.Info("unfreeze token failed", "reason", "no enough frozen tokens to unfreeze")
		return sdk.ErrInsufficientCoins("do not have enough token to unfreeze").Result()
	}

	newFrozenTokens := account.GetFrozenCoins().Minus(sdk.Coins{{Denom: symbol, Amount: unfreezeAmount}})
	newFreeTokens := account.GetCoins().Plus(sdk.Coins{{Denom: symbol, Amount: unfreezeAmount}})
	account.SetFrozenCoins(newFrozenTokens)
	account.SetCoins(newFreeTokens)
	accKeeper.SetAccount(ctx, account)
	logger.Debug("finish unfreezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}

func handleFreezeMiniToken(ctx sdk.Context, miniTokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper, msg FreezeMsg) sdk.Result {
	freezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "mini-token", "symbol", symbol, "amount", freezeAmount, "addr", msg.From)
	errLogMsg := "freeze token failed"
	_, err := miniTokenMapper.GetMiniToken(ctx, symbol)
	if err != nil {
		logger.Info(errLogMsg, "reason", "symbol not exist")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	coins := keeper.GetCoins(ctx, msg.From)
	if coins.AmountOf(symbol) < freezeAmount {
		logger.Info(errLogMsg, "reason", "no enough free tokens to freeze")
		return sdk.ErrInsufficientCoins("do not have enough token to freeze").Result()
	}

	useAllBalance := coins.AmountOf(symbol) == freezeAmount

	if msg.Amount <= 0 || (!useAllBalance && (msg.Amount < common.MiniTokenMinTotalSupply)) {
		logger.Info(errLogMsg, "reason", "freeze amount doesn't reach the min supply")
		return sdk.ErrInvalidCoins(fmt.Sprintf("freeze amount is too small, the min amount is %d or total account balance",
			common.MiniTokenMinTotalSupply)).Result()
	}

	account := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	newFrozenTokens := account.GetFrozenCoins().Plus(sdk.Coins{{Denom: symbol, Amount: freezeAmount}})
	newFreeTokens := account.GetCoins().Minus(sdk.Coins{{Denom: symbol, Amount: freezeAmount}})
	account.SetFrozenCoins(newFrozenTokens)
	account.SetCoins(newFreeTokens)
	accKeeper.SetAccount(ctx, account)
	logger.Info("finish freezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}

func handleUnfreezeMiniToken(ctx sdk.Context, miniTokenMapper store.Mapper, accKeeper auth.AccountKeeper, keeper bank.Keeper, msg UnfreezeMsg) sdk.Result {
	unfreezeAmount := msg.Amount
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "mini-token", "symbol", symbol, "amount", unfreezeAmount, "addr", msg.From)
	account := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	frozenAmount := account.GetFrozenCoins().AmountOf(symbol)
	useAllFrozenBalance := frozenAmount == unfreezeAmount
	errLogMsg := "unfreeze token failed"

	_, err := miniTokenMapper.GetMiniToken(ctx, symbol)
	if err != nil {
		logger.Info(errLogMsg, "reason", "symbol not exist")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	if unfreezeAmount <= 0 || (!useAllFrozenBalance && (unfreezeAmount < common.MiniTokenMinTotalSupply)) {
		logger.Info(errLogMsg, "reason", "unfreeze amount doesn't reach the min supply")
		return sdk.ErrInvalidCoins(fmt.Sprintf("freeze amount is too small, the min amount is %d or total frozen balance",
			common.MiniTokenMinTotalSupply)).Result()
	}

	if frozenAmount < unfreezeAmount {
		logger.Info(errLogMsg, "reason", "no enough frozen tokens to unfreeze")
		return sdk.ErrInsufficientCoins("do not have enough token to unfreeze").Result()
	}

	newFrozenTokens := account.GetFrozenCoins().Minus(sdk.Coins{{Denom: symbol, Amount: unfreezeAmount}})
	newFreeTokens := account.GetCoins().Plus(sdk.Coins{{Denom: symbol, Amount: unfreezeAmount}})

	account.SetFrozenCoins(newFrozenTokens)
	account.SetCoins(newFreeTokens)
	accKeeper.SetAccount(ctx, account)
	logger.Debug("finish unfreezing token", "NewFrozenToken", newFrozenTokens, "NewFreeTokens", newFreeTokens)
	return sdk.Result{}
}
