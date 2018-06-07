package freeze

import (
	"reflect"
	"strings"

	"github.com/BiJie/BinanceChain/plugins/tokens/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

func NewHandler(tokenMapper store.Mapper, keeper bank.CoinKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(Msg); ok {
			return handleFreezeToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleFreezeToken(ctx sdk.Context, tokenMapper store.Mapper, keeper bank.CoinKeeper, msg Msg) sdk.Result {
	freezeAmount := msg.Amount
	if freezeAmount <= 0 {
		return sdk.Result{Code: sdk.CodeInsufficientFunds}
	}

	symbol := strings.ToUpper(msg.Symbol)
	exists := tokenMapper.Exists(ctx, symbol)
	if !exists {
		return sdk.Result{Code: sdk.CodeInvalidCoins}
	}

	// TODO: the third param can be removed...
	coins := keeper.GetCoins(ctx, msg.Owner, nil)
	var theToken sdk.Coin
	for _, coin := range coins {
		if coin.Denom == symbol {
			theToken = coin
			break
		}
	}

	if theToken.Amount < freezeAmount {
		return sdk.Result{Code: sdk.CodeInsufficientCoins}
	}

	_, sdkError := keeper.SubtractCoins(ctx, msg.Owner, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: freezeAmount}))
	// TODO: update freeze
	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}
