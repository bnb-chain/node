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
	if keeper.PairMapper.Exists(ctx, msg.BaseAsset, msg.QuoteAsset) || keeper.PairMapper.Exists(ctx, msg.QuoteAsset, msg.BaseAsset) {
		return sdk.ErrInvalidCoins("trading pair exists").Result()
	}

	baseToken, err := tokenMapper.GetToken(ctx, msg.BaseAsset)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !baseToken.IsOwner(msg.From) {
		return sdk.ErrUnauthorized("only the owner of the token can list the token").Result()
	}

	if !tokenMapper.Exists(ctx, msg.QuoteAsset) {
		return sdk.ErrInvalidCoins("quote token does not exist").Result()
	}

	pair := types.NewTradingPair(msg.BaseAsset, msg.QuoteAsset, msg.InitPrice)
	err = keeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	if !ctx.IsCheckTx() { // only add engine during DeliverTx
		keeper.AddEngine(pair)
	}

	return sdk.Result{}
}
