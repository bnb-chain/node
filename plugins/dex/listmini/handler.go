package listmini

import (
	"fmt"
	"reflect"

	"github.com/binance-chain/node/common/log"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/plugins/minitokens"
	"github.com/binance-chain/node/plugins/tokens"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewHandler initialises dex message handlers
func NewHandler(keeper *order.Keeper, miniTokenMapper minitokens.MiniTokenMapper, tokenMapper tokens.Mapper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case ListMiniMsg:
			return handleList(ctx, keeper, miniTokenMapper, tokenMapper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleList(ctx sdk.Context, keeper *order.Keeper, miniTokenMapper minitokens.MiniTokenMapper, tokenMapper tokens.Mapper,
	msg ListMiniMsg) sdk.Result {
	if !sdk.IsUpgrade(upgrade.BEP8) {
		return sdk.ErrInternal(fmt.Sprint("list miniToken is not supported at current height")).Result()
	}

	if err := keeper.CanListTradingPair(ctx, msg.BaseAssetSymbol, msg.QuoteAssetSymbol); err != nil {
		//TODO use miniTradingPair
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

	if common.NativeTokenSymbol != msg.QuoteAssetSymbol { //todo permit BUSD
		return sdk.ErrInvalidCoins("quote token: " + err.Error()).Result()
	}

	var lotSize int64
	if sdk.IsUpgrade(upgrade.LotSizeOptimization) {
		lotSize = keeper.DetermineLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice)
	} else {
		lotSize = utils.CalcLotSize(msg.InitPrice)
	}
	pair := types.NewTradingPairWithLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice, lotSize)
	err = keeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	// this is done in memory! we must not run this block in checktx or simulate!
	if ctx.IsDeliverTx() { // only add engine during DeliverTx
		keeper.AddEngine(pair)
		log.With("module", "dex").Info("List new mini-token Pair and created new match engine", "pair", pair)
	}

	return sdk.Result{}
}
