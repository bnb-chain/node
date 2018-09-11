package burn

import (
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/account"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, keeper account.Keeper) types.Handler {
	return func(ctx types.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(Msg); ok {
			return handleBurnToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleBurnToken(ctx types.Context, tokenMapper store.Mapper, keeper account.Keeper, msg Msg) sdk.Result {
	burnAmount := msg.Amount
	if burnAmount <= 0 {
		return sdk.ErrInsufficientCoins("burn amount should be greater than 0").Result()
	}

	symbol := strings.ToUpper(msg.Symbol)
	token, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized("only the owner of the token can burn the token").Result()
	}

	coins := keeper.GetCoins(ctx, token.Owner)
	if coins.AmountOf(symbol).Int64() < burnAmount || token.TotalSupply < burnAmount {
		return sdk.ErrInsufficientCoins("do not have enough token to burn").Result()
	}

	_, _, sdkError := keeper.SubtractCoins(ctx, token.Owner, append((sdk.Coins)(nil), sdk.Coin{Denom: symbol, Amount: sdk.NewInt(burnAmount)}))
	if sdkError != nil {
		return sdkError.Result()
	}

	err = tokenMapper.UpdateTotalSupply(ctx, symbol, token.TotalSupply-burnAmount)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	return sdk.Result{}
}
