package burn

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
			return handleBurnToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleBurnToken(ctx sdk.Context, tokenMapper store.Mapper, keeper bank.CoinKeeper, msg Msg) sdk.Result {
	burnAmount := msg.Amount
	if burnAmount <= 0 {
		return sdk.Result{Code: sdk.CodeInsufficientFunds}
	}

	symbol := strings.ToUpper(msg.Symbol)
	_, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	// TODO: the third param can be removed...
	// TODO: add a function to get balance of the specific token
	coins := keeper.GetCoins(ctx, msg.Owner, nil)
	found := false
	var theToken sdk.Coin
	for _, coin := range coins {
		if coin.Denom == symbol {
			theToken = coin
			found = true
			break
		}
	}

	if !found || theToken.Amount < burnAmount {
		return sdk.ErrInsufficientCoins("do not have enough token to burn").Result()
	}

	_, sdkError := keeper.SubtractCoins(ctx, msg.Owner, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: burnAmount}))
	// TODO: update supply

	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}
