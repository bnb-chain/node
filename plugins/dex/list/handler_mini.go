package list

import (
	"github.com/binance-chain/node/common/log"
	ctypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/tokens"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func handleListMini(ctx sdk.Context, dexKeeper *order.DexKeeper, tokenMapper tokens.Mapper,
	msg types.ListMiniMsg) sdk.Result {

	// before BEP70 upgraded, we only support listing mini token against NativeToken
	if sdk.IsUpgrade(upgrade.BEP70) {
		if ctypes.NativeTokenSymbol != msg.QuoteAssetSymbol && order.BUSDSymbol != msg.QuoteAssetSymbol {
			return sdk.ErrInvalidCoins("quote token is not valid ").Result()
		}
	} else {
		if ctypes.NativeTokenSymbol != msg.QuoteAssetSymbol {
			return sdk.ErrInvalidCoins("quote token is not valid ").Result()
		}
	}

	if err := dexKeeper.CanListTradingPair(ctx, msg.BaseAssetSymbol, msg.QuoteAssetSymbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

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
		log.With("module", "dex").Info("List new mini-token Pair and created new match engine", "pair", pair)
	}

	return sdk.Result{}
}
