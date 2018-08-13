package freeze

import (
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) tx.Handler {
	return func(ctx sdk.Context, msg tx.Msg) sdk.Result {
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
	if freezeAmount <= 0 {
		return sdk.ErrInsufficientCoins("freeze amount should be greater than 0").Result()
	}

	symbol := strings.ToUpper(msg.Symbol)
	coins := keeper.GetCoins(ctx, msg.From)
	if coins.AmountOf(symbol).Int64() < freezeAmount {
		return sdk.ErrInsufficientCoins("do not have enough token to freeze").Result()
	}

	_, _, sdkError := keeper.SubtractCoins(ctx, msg.From, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: sdk.NewInt(freezeAmount)}))
	if sdkError != nil {
		return sdkError.Result()
	}

	updateFrozenOfAccount(ctx, accountMapper, msg.From, symbol, freezeAmount)
	return sdk.Result{}
}

func handleUnfreezeToken(ctx sdk.Context, tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper, msg UnfreezeMsg) sdk.Result {
	unfreezeAmount := msg.Amount
	if unfreezeAmount <= 0 {
		return sdk.ErrInsufficientCoins("unfreeze amount should be greater than 0").Result()
	}

	symbol := strings.ToUpper(msg.Symbol)
	account := accountMapper.GetAccount(ctx, msg.From).(types.NamedAccount)
	frozenAmount := account.GetFrozenCoins().AmountOf(symbol).Int64()
	if frozenAmount < unfreezeAmount {
		return sdk.ErrInsufficientCoins("do not have enough token to unfreeze").Result()
	}

	account.SetFrozenCoins(account.GetFrozenCoins().Minus(append(sdk.Coins{}, sdk.Coin{Denom: symbol, Amount: sdk.NewInt(unfreezeAmount)})))
	accountMapper.SetAccount(ctx, account)

	_, _, sdkError := keeper.AddCoins(ctx, msg.From, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: sdk.NewInt(unfreezeAmount)}))

	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}

func updateFrozenOfAccount(ctx sdk.Context, accountMapper auth.AccountMapper, address sdk.AccAddress, symbol string, frozenAmount int64) {
	account := accountMapper.GetAccount(ctx, address).(types.NamedAccount)
	account.SetFrozenCoins(account.GetFrozenCoins().Plus(append(sdk.Coins{}, sdk.Coin{Denom: symbol, Amount: sdk.NewInt(frozenAmount)})))
	accountMapper.SetAccount(ctx, account)
}
