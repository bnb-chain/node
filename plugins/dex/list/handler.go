package list

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

func NewHandler(keeper order.Keeper, tokenMapper tokens.Mapper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case Msg:
			return handleList(ctx, keeper, tokenMapper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleList(ctx sdk.Context, keeper order.Keeper, tokenMapper tokens.Mapper, msg Msg) sdk.Result {
	if keeper.PairMapper.Exists(ctx, msg.Symbol, msg.QuoteSymbol) || keeper.PairMapper.Exists(ctx, msg.QuoteSymbol, msg.Symbol) {
		return sdk.ErrInvalidCoins("trading pair exists").Result()
	}

	tradeToken, err := tokenMapper.GetToken(ctx, msg.Symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !tradeToken.IsOwner(msg.From) {
		return sdk.ErrUnauthorized("only the owner of the token can list the token").Result()
	}

	if !tokenMapper.Exists(ctx, msg.QuoteSymbol) {
		return sdk.ErrInvalidCoins("quote token does not exist").Result()
	}

	pair := types.NewTradingPair(msg.Symbol, msg.QuoteSymbol, msg.InitPrice)
	err = keeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	if !ctx.IsCheckTx() { // only add engine during DeliverTx
		keeper.AddEngine(pair)
		ctx.Logger().Info("Created new match engine", "pair", pair)
	}

	return sdk.Result{}
}
