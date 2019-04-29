package dex

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/app/pub"
	bnclog "github.com/binance-chain/node/common/log"
	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/utils"
	tkstore "github.com/binance-chain/node/plugins/tokens/store"
)

const AbciQueryPrefix = "dex"
const DelistDelayedDays = 7

// InitPlugin initializes the dex plugin.
func InitPlugin(
	appp app.ChainApp, keeper *DexKeeper, tokenMapper tkstore.Mapper, accMapper auth.AccountKeeper, govKeeper gov.Keeper,
) {
	cdc := appp.GetCodec()

	// add msg handlers
	for route, handler := range Routes(cdc, keeper, tokenMapper, accMapper, govKeeper) {
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
func EndBreatheBlock(ctx sdk.Context, dexKeeper *DexKeeper, height int64, blockTime time.Time) {
	logger := bnclog.With("module", "dex")
	logger.Info("Update tick size / lot size")
	dexKeeper.UpdateTickSizeAndLotSize(ctx)
	logger.Info("Expire stale orders")
	if dexKeeper.CollectOrderInfoForPublish {
		pub.ExpireOrdersForPublish(dexKeeper, ctx, blockTime)
	} else {
		dexKeeper.ExpireOrders(ctx, blockTime, nil)
	}
	logger.Info("Mark BreathBlock", "blockHeight", height)
	dexKeeper.MarkBreatheBlock(ctx, height, blockTime)
	logger.Info("Save Orderbook snapshot", "blockHeight", height)
	if _, err := dexKeeper.SnapShotOrderBook(ctx, height); err != nil {
		logger.Error("Failed to snapshot order book", "blockHeight", height, "err", err)
	}
	return
}

func delistTradingPairs(ctx sdk.Context, govKeeper gov.Keeper, dexKeeper *DexKeeper, blockTime time.Time) {
	symbolsToDelist := getSymbolsToDelist(ctx, govKeeper, blockTime)
	if len(symbolsToDelist) == 0 {
		return
	}

	for _, symbol := range symbolsToDelist {
		dexKeeper.DelistTradingPair(ctx, symbol)
	}
}

func getSymbolsToDelist(ctx sdk.Context, govKeeper gov.Keeper, blockTime time.Time) []string {
	symbols := make([]string, 0)
	govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, -1, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeDelistTradingPair {
			passedTime := proposal.GetVotingStartTime().Add(proposal.GetVotingPeriod())
			if passedTime.Add((DelistDelayedDays-1)*24*time.Hour).Before(blockTime) &&
				passedTime.Add(DelistDelayedDays*24*time.Hour).After(blockTime) {
				var delistParam gov.DelistTradingPairParams
				err := json.Unmarshal([]byte(proposal.GetDescription()), &delistParam)
				if err != nil {
					panic(fmt.Errorf("illegal delist params in proposal, params=%s", proposal.GetDescription()))
				}
				symbol := utils.Assets2TradingPair(strings.ToUpper(delistParam.BaseAssetSymbol), strings.ToUpper(delistParam.QuoteAssetSymbol))
				symbols = append(symbols, symbol)
			}
		}
		return false
	})
	return symbols
}
