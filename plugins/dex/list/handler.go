package list

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

func NewHandler(pairMapper store.TradingPairMapper, tokenMapper tokens.Mapper) tx.Handler {
	return func(ctx sdk.Context, msg tx.Msg) sdk.Result {
		switch msg := msg.(type) {
		case Msg:
			return handleList(ctx, pairMapper, tokenMapper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleList(ctx sdk.Context, pairMapper store.TradingPairMapper, tokenMapper tokens.Mapper, msg Msg) sdk.Result {
	if pairMapper.Exists(ctx, msg.Symbol, msg.QuoteSymbol) || pairMapper.Exists(ctx, msg.QuoteSymbol, msg.Symbol) {
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
	err = pairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	return sdk.Result{}
}
