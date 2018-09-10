package issue

import (
	"fmt"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx types.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(Msg); ok {
			return handleIssueToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleIssueToken(ctx types.Context, tokenMapper store.Mapper, keeper bank.Keeper, msg Msg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)
	exists := tokenMapper.Exists(ctx, symbol)
	if exists {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	token := types.NewToken(msg.Name, symbol, msg.TotalSupply, msg.From)
	err := tokenMapper.NewToken(ctx, token)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	_, _, sdkError := keeper.AddCoins(ctx, token.Owner, append((sdk.Coins)(nil), sdk.Coin{Denom: token.Symbol, Amount: sdk.NewInt(token.TotalSupply)}))
	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}
