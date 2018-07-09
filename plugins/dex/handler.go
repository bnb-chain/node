package dex

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// NewHandler - returns a handler for dex type messages.
func NewHandler(k Keeper, accountMapper auth.AccountMapper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case NewOrderMsg:
			return handleNewOrder(ctx, k, accountMapper, msg)
		case CancelOrderMsg:
			return handleCancelOrder(ctx, k, accountMapper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

//TODO: duplicated with plugins/tokens/freeze/handler.go
func updateLockedOfAccount(ctx sdk.Context, accountMapper auth.AccountMapper, address sdk.Address, symbol string, lockedAmount int64) {
	account := accountMapper.GetAccount(ctx, address).(types.NamedAccount)
	account.SetLockedCoins(account.GetLockedCoins().Plus(append(sdk.Coins{}, sdk.Coin{Denom: symbol, Amount: lockedAmount})))
	accountMapper.SetAccount(ctx, account)
}

func handleNewOrder(ctx sdk.Context, keeper Keeper, accountMapper auth.AccountMapper, msg NewOrderMsg) sdk.Result {
	//TODO: the below is mostly copied from FreezeToken. It should be rewritten once "locked" becomes a field on account
	freezeAmount := msg.Quantity
	if freezeAmount <= 0 {
		return sdk.ErrInsufficientCoins("freeze amount should be greater than 0").Result()
	}
	tradeCcy, quoteCcy, _ := utils.TradeSymbol2Ccy(msg.Symbol)
	var symbolToLock string
	if msg.Side == Side.BUY {
		symbolToLock = strings.ToUpper(tradeCcy)
	} else {
		symbolToLock = strings.ToUpper(quoteCcy)
	}
	coins := keeper.ck.GetCoins(ctx, msg.Sender)
	if coins.AmountOf(symbolToLock) < freezeAmount {
		return sdk.ErrInsufficientCoins("do not have enough token to freeze").Result()
	}

	_, _, sdkError := keeper.ck.SubtractCoins(ctx, msg.Sender, append((sdk.Coins)(nil), sdk.Coin{Denom: symbolToLock, Amount: freezeAmount}))
	if sdkError != nil {
		return sdkError.Result()
	}

	updateLockedOfAccount(ctx, accountMapper, msg.Sender, symbolToLock, freezeAmount)
	return sdk.Result{}
}

// Handle CancelOffer -
func handleCancelOrder(ctx sdk.Context, k Keeper, accountMapper auth.AccountMapper, msg CancelOrderMsg) sdk.Result {
	return sdk.Result{}
}
