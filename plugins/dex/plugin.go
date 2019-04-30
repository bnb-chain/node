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
const DaysToSearchForDelist = 60 // it is a approximate number to search for proposal, for the precise number is stored in db

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
func EndBreatheBlock(ctx sdk.Context, dexKeeper *DexKeeper, govKeeper gov.Keeper, height int64, blockTime time.Time) {
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
	delistTradingPairs(ctx, govKeeper, dexKeeper, blockTime)
	logger.Info("Delist trading pairs", "blockHeight", height)
	return
}

func delistTradingPairs(ctx sdk.Context, govKeeper gov.Keeper, dexKeeper *DexKeeper, blockTime time.Time) {
	logger := bnclog.With("module", "dex")
	symbolsToDelist := getSymbolsToDelist(ctx, govKeeper, blockTime)

	for _, symbol := range symbolsToDelist {
		logger.Info("Delist trading pair", "symbol", symbol)
		dexKeeper.DelistTradingPair(ctx, symbol)
	}
}

func getSymbolsToDelist(ctx sdk.Context, govKeeper gov.Keeper, blockTime time.Time) []string {
	symbols := make([]string, 0)
	govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, -1, true, func(proposal gov.Proposal) bool {
		// we do not need to search for all proposals
		if proposal.GetSubmitTime().Add(DaysToSearchForDelist * 24 * time.Hour).Before(blockTime) {
			return true
		}

		if proposal.GetProposalType() == gov.ProposalTypeDelistTradingPair {
			var delistParam gov.DelistTradingPairParams
			err := json.Unmarshal([]byte(proposal.GetDescription()), &delistParam)
			if err != nil {
				panic(fmt.Errorf("illegal delist params in proposal, params=%s", proposal.GetDescription()))
			}

			passedTime := proposal.GetVotingStartTime().Add(proposal.GetVotingPeriod())
			timeToCompare := passedTime.Add(time.Duration(delistParam.DelayedDays) * 24 * time.Hour)
			if timeToCompare.Before(blockTime) && timeToCompare.Add(24*time.Hour).After(blockTime) {
				symbol := utils.Assets2TradingPair(strings.ToUpper(delistParam.BaseAssetSymbol), strings.ToUpper(delistParam.QuoteAssetSymbol))
				symbols = append(symbols, symbol)
			}
		}
		return false
	})
	return symbols
}
