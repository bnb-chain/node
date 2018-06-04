package tokens

import (
	"fmt"
	"math/big"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

func NewHandler(tokenMapper Mapper, keeper bank.CoinKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case IssueMsg:
			return handleIssueToken(ctx, tokenMapper, keeper, msg)
		default:
			errMsg := "Unreconized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleIssueToken(ctx sdk.Context, tokenMapper Mapper, keeper bank.CoinKeeper, msg IssueMsg) sdk.Result {
	// TODO: validate if the coin's symbol exists
	fmt.Println("handle IssueToken")

	token := msg.Token
	exists := tokenMapper.Exists(ctx, token.Symbol)
	if exists {
		return sdk.Result{ Code:sdk.CodeInvalidCoins }
	}

	err := tokenMapper.NewToken(ctx, token)
	if err != nil {
		return sdk.Result{ Code:sdk.CodeInvalidCoins }
	}

	// amount = supply * 10^decimals
	amount := new(big.Int)
	amount.Mul(amount.Exp(big.NewInt(10), token.Decimals, nil), token.Supply)
	// TODO: need to fix Coin#Amount type to big.Int
	_, sdkError := keeper.AddCoins(ctx, msg.Owner, append((sdk.Coins)(nil), sdk.Coin{Denom: token.Symbol, Amount: amount.Int64()}))
	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}
