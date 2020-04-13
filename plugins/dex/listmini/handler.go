package listmini

import (
	"fmt"
	"reflect"

	"github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/plugins/minitokens"
	"github.com/binance-chain/node/plugins/tokens"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewHandler initialises dex message handlers
func NewHandler(miniKeeper *order.MiniKeeper, miniTokenMapper minitokens.MiniTokenMapper, tokenMapper tokens.Mapper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case ListMiniMsg:
			return handleList(ctx, miniKeeper, miniTokenMapper, tokenMapper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleList(ctx sdk.Context, miniKeeper *order.MiniKeeper, miniTokenMapper minitokens.MiniTokenMapper, tokenMapper tokens.Mapper,
	msg ListMiniMsg) sdk.Result {
	if !sdk.IsUpgrade(upgrade.BEP8) {
		return sdk.ErrInternal(fmt.Sprint("list mini-token is not supported at current height")).Result()
	}

	if err := miniKeeper.CanListTradingPair(ctx, msg.BaseAssetSymbol, msg.QuoteAssetSymbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	baseToken, err := miniTokenMapper.GetToken(ctx, msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	quoteToken, err := tokenMapper.GetToken(ctx, msg.QuoteAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !baseToken.IsOwner(msg.From) && !quoteToken.IsOwner(msg.From) {
		return sdk.ErrUnauthorized("only the owner of the base asset or quote asset can list the trading pair").Result()
	}

	if !tokenMapper.Exists(ctx, msg.QuoteAssetSymbol) {
		return sdk.ErrInvalidCoins("quote token does not exist").Result()
	}

	var lotSize int64
	if sdk.IsUpgrade(upgrade.LotSizeOptimization) {
		lotSize = miniKeeper.DetermineLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice)
	} else {
		lotSize = utils.CalcLotSize(msg.InitPrice)
	}
	pair := types.NewTradingPairWithLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice, lotSize)
	err = miniKeeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	// this is done in memory! we must not run this block in checktx or simulate!
	if ctx.IsDeliverTx() { // only add engine during DeliverTx
		miniKeeper.AddEngine(pair)
		log.With("module", "dex").Info("List new mini-token Pair and created new match engine", "pair", pair)
	}

	return sdk.Result{}
}
