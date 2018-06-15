package burn

import (
	"math"
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
		return sdk.ErrInsufficientCoins("burn amount should be greater than 0").Result()
	}

	symbol := strings.ToUpper(msg.Symbol)
	token, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	innerBurnAmount := int64(math.Pow10(int(token.Decimal))) * burnAmount

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized("only the owner of the token can burn the token").Result()
	}

	// the token owner burns the token from the token account
	// TODO: the third param can be removed...
	coins := keeper.GetCoins(ctx, token.Address, nil)
	if coins.AmountOf(symbol) < innerBurnAmount || token.Supply < burnAmount {
		return sdk.ErrInsufficientCoins("do not have enough token to burn").Result()
	}

	_, sdkError := keeper.SubtractCoins(ctx, token.Address, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: innerBurnAmount}))
	tokenMapper.UpdateTokenSupply(ctx, symbol, token.Supply-burnAmount)

	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}
