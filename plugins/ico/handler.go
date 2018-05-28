package ico

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"reflect"
)

func NewHandler(keeper bank.CoinKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case IssueMsg:
			return handleIssueToken(ctx, keeper, msg)
		default:
			errMsg := "Unreconized ico msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleIssueToken(ctx sdk.Context, keeper bank.CoinKeeper, msg IssueMsg) sdk.Result {
	// TODO: validate if the coin's symbol exists
	coins := (sdk.Coins)(nil)
	_, err := keeper.AddCoins(ctx, msg.Banker, append(coins, msg.Coin))
	if err != nil {
		return err.Result();
	}

	return sdk.Result{}
}
