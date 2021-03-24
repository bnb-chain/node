package list

import (
	"fmt"

	"github.com/binance-chain/node/common/log"
	ctypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/tokens"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func handleListGrowthMarket(ctx sdk.Context, dexKeeper *order.DexKeeper, tokenMapper tokens.Mapper,
	msg types.ListGrowthMarketMsg) sdk.Result {

	if ctypes.NativeTokenSymbol != msg.QuoteAssetSymbol && order.BUSDSymbol != msg.QuoteAssetSymbol {
		return sdk.ErrInvalidCoins("quote token is not valid ").Result()
	}

	if ctypes.NativeTokenSymbol == msg.QuoteAssetSymbol {
		if pair, err := dexKeeper.PairMapper.GetTradingPair(ctx, msg.BaseAssetSymbol, order.BUSDSymbol); err == nil {
			log.Info(fmt.Sprintf("%s", pair)) // todo remove this log
			// todo if pair type is main market, return error message: One token can only be listed on one market
		}

	} else if order.BUSDSymbol != msg.QuoteAssetSymbol {
		if pair, err := dexKeeper.PairMapper.GetTradingPair(ctx, msg.BaseAssetSymbol, ctypes.NativeTokenSymbol); err == nil {
			log.Info(fmt.Sprintf("%s", pair)) // todo remove this log
			// todo if pair type is main market, return error message: One token can only be listed on one market
		}
	} else {
		return sdk.ErrInvalidCoins("quote token is not valid ").Result()
	}

	if err := dexKeeper.CanListTradingPair(ctx, msg.BaseAssetSymbol, msg.QuoteAssetSymbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	// todo check if exists a trading pair taking msg.BaseAsset as base quote in main market

	baseToken, err := tokenMapper.GetToken(ctx, msg.BaseAssetSymbol)
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

	lotSize := dexKeeper.DetermineLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice)

	pair := types.NewTradingPairWithLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice, lotSize)
	err = dexKeeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	// this is done in memory! we must not run this block in checktx or simulate!
	if ctx.IsDeliverTx() { // only add engine during DeliverTx
		dexKeeper.AddEngine(pair)
		log.With("module", "dex").Info("List new pair on growth market and created new match engine", "pair", pair)
	}

	return sdk.Result{}
}
