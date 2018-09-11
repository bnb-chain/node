package freeze

import (
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/account"
	"github.com/BiJie/BinanceChain/common/types"
)

func NewHandler(accountMapper account.Mapper, keeper account.Keeper) types.Handler {
	return func(ctx types.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case FreezeMsg:
			return handleFreezeToken(ctx, accountMapper, keeper, msg)
		case UnfreezeMsg:
			return handleUnfreezeToken(ctx, accountMapper, keeper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleFreezeToken(ctx types.Context, accountMapper account.Mapper, keeper account.Keeper, msg FreezeMsg) sdk.Result {
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

func handleUnfreezeToken(ctx types.Context, accountMapper account.Mapper, keeper account.Keeper, msg UnfreezeMsg) sdk.Result {
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

func updateFrozenOfAccount(ctx types.Context, accountMapper account.Mapper, address sdk.AccAddress, symbol string, frozenAmount int64) {
	account := accountMapper.GetAccount(ctx, address).(types.NamedAccount)
	account.SetFrozenCoins(account.GetFrozenCoins().Plus(append(sdk.Coins{}, sdk.Coin{Denom: symbol, Amount: sdk.NewInt(frozenAmount)})))
	accountMapper.SetAccount(ctx, account)
}
