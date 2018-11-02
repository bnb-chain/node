package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/app/pub"
	bnclog "github.com/BiJie/BinanceChain/common/log"
	app "github.com/BiJie/BinanceChain/common/types"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
)

const AbciQueryPrefix = "dex"

// InitPlugin initializes the dex plugin.
func InitPlugin(
	appp app.ChainApp, keeper *DexKeeper, tokenMapper tkstore.Mapper, accMapper auth.AccountMapper,
) {
	cdc := appp.GetCodec()

	// add msg handlers
	for route, handler := range Routes(cdc, keeper, tokenMapper, accMapper) {
		appp.GetRouter().AddRoute(route, handler)
	}

	// add abci handlers
	handler := createQueryHandler(keeper)
	appp.RegisterQueryHandler(AbciQueryPrefix, handler)
}

func createQueryHandler(keeper *DexKeeper) app.AbciQueryHandler {
	return createAbciQueryHandler(keeper)
}

// EndBreatheBlock processes the breathe block lifecycle event.
func EndBreatheBlock(ctx sdk.Context, accountMapper auth.AccountMapper, dexKeeper *DexKeeper, height, blockTime int64) {
	logger := bnclog.With("module", "dex")
	logger.Info("Update tick size / lot size")
	updateTickSizeAndLotSize(ctx, dexKeeper)
	// TODO: update fee/rate
	logger.Info("Expire stale orders")
	if dexKeeper.CollectOrderInfoForPublish {
		pub.ExpireOrdersForPublish(dexKeeper, ctx, blockTime)
	} else {
		dexKeeper.ExpireOrders(ctx, blockTime, nil, nil)
	}
	logger.Info("Mark BreathBlock", "blockHeight", height)
	dexKeeper.MarkBreatheBlock(ctx, height, blockTime)
	logger.Info("Save Orderbook snapshot", "blockHeight", height)
	if _, err := dexKeeper.SnapShotOrderBook(ctx, height); err != nil {
		logger.Error("Failed to snapshot order book", "blockHeight", height, "err", err)
	}
}

func updateTickSizeAndLotSize(ctx sdk.Context, dexKeeper *DexKeeper) {
	tradingPairs := dexKeeper.PairMapper.ListAllTradingPairs(ctx)

	for _, pair := range tradingPairs {
		_, lastPrice := dexKeeper.GetLastTradesForPair(pair.GetSymbol())
		if lastPrice == 0 {
			continue
		}
		_, lotSize := dexKeeper.PairMapper.UpdateTickSizeAndLotSize(ctx, pair, lastPrice)
		dexKeeper.UpdateLotSize(pair.GetSymbol(), lotSize)
	}
}
