package list

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

// NewHandler initialises dex message handlers
func NewHandler(keeper *order.Keeper, tokenMapper tokens.Mapper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case ListMsg:
			return handleList(ctx, keeper, tokenMapper, msg, simulate)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleList(
	ctx sdk.Context, keeper *order.Keeper, tokenMapper tokens.Mapper, msg ListMsg, simulate bool,
) sdk.Result {
	if keeper.PairMapper.Exists(ctx, msg.BaseAssetSymbol, msg.QuoteAssetSymbol) || keeper.PairMapper.Exists(ctx, msg.QuoteAssetSymbol, msg.BaseAssetSymbol) {
		return sdk.ErrInvalidCoins("trading pair exists").Result()
	}

	baseToken, err := tokenMapper.GetToken(ctx, msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !baseToken.IsOwner(msg.From) {
		return sdk.ErrUnauthorized("only the owner of the token can list the token").Result()
	}

	if !tokenMapper.Exists(ctx, msg.QuoteAssetSymbol) {
		return sdk.ErrInvalidCoins("quote token does not exist").Result()
	}

	pair := types.NewTradingPair(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice)
	err = keeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	// this is done in memory! we must not run this block in checktx or simulate!
	if !ctx.IsCheckTx() && !simulate { // only add engine during DeliverTx
		keeper.AddEngine(pair)
		log.With("module", "dex").Info("List new Pair and created new match engine", "pair", pair)
	}

	return sdk.Result{}
}
