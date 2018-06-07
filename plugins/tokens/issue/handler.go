package issue

import (
	"math/big"
	"reflect"
	"strings"

	"github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/cosmos/cosmos-sdk/x/bank"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewHandler(tokenMapper store.Mapper, keeper bank.CoinKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(Msg); ok {
			return handleIssueToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleIssueToken(ctx sdk.Context, tokenMapper store.Mapper, keeper bank.CoinKeeper, msg Msg) sdk.Result {
	token := msg.Token
	token.Symbol = strings.ToUpper(token.Symbol)
	exists := tokenMapper.Exists(ctx, token.Symbol)
	if exists {
		return sdk.Result{Code: sdk.CodeInvalidCoins}
	}

	err := tokenMapper.NewToken(ctx, token)
	if err != nil {
		return sdk.Result{Code: sdk.CodeInvalidCoins}
	}

	// amount = supply * 10^decimals
	amount := new(big.Int)
	// TODO: maybe need to wrap the big.Int methods
	amount.Mul(amount.Exp(big.NewInt(10), token.Decimal.ToBigInt(), nil), token.Supply.ToBigInt())
	// TODO: need to fix Coin#Amount type to big.Int
	_, sdkError := keeper.AddCoins(ctx, msg.Owner, append((sdk.Coins)(nil), sdk.Coin{Denom: token.Symbol, Amount: amount.Int64()}))
	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}
