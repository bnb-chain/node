package dex

import (
	"encoding/json"
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

const DexAbciQueryPrefix = "dex"
const DexMiniAbciQueryPrefix = "dex-mini"
const DelayedDaysForDelist = 3

// InitPlugin initializes the dex plugin.
func InitPlugin(
	appp app.ChainApp, dexKeeper *DexKeeper, tokenMapper tkstore.Mapper, accMapper auth.AccountKeeper, govKeeper gov.Keeper,
) {
	cdc := appp.GetCodec()

	// add msg handlers
	for route, handler := range Routes(cdc, dexKeeper, tokenMapper, accMapper, govKeeper) {
		appp.GetRouter().AddRoute(route, handler)
	}

	// add abci handlers
	dexHandler := createQueryHandler(dexKeeper, DexAbciQueryPrefix)
	appp.RegisterQueryHandler(DexAbciQueryPrefix, dexHandler)
	//dex mini handler
	dexMiniHandler := createQueryHandler(dexKeeper, DexMiniAbciQueryPrefix)
	appp.RegisterQueryHandler(DexMiniAbciQueryPrefix, dexMiniHandler)
}

func createQueryHandler(keeper *DexKeeper, abciQueryPrefix string) app.AbciQueryHandler {
	return createAbciQueryHandler(keeper, abciQueryPrefix)
}

// EndBreatheBlock processes the breathe block lifecycle event.
func EndBreatheBlock(ctx sdk.Context, dexKeeper *DexKeeper, govKeeper gov.Keeper, height int64, blockTime time.Time) {
	logger := bnclog.With("module", "dex")

	logger.Info("Delist trading pairs", "blockHeight", height)
	delistTradingPairs(ctx, govKeeper, dexKeeper, blockTime)

	logger.Info("Update tick size / lot size")
	dexKeeper.UpdateTickSizeAndLotSize(ctx)

	logger.Info("Expire stale orders")
	if dexKeeper.ShouldPublishOrder() {
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
	logger := bnclog.With("module", "dex")
	symbolsToDelist := getSymbolsToDelist(ctx, govKeeper, blockTime)

	for _, symbol := range symbolsToDelist {
		logger.Info("Delist trading pair", "symbol", symbol)
		baseAsset, quoteAsset := utils.TradingPair2AssetsSafe(symbol)
		err := dexKeeper.CanDelistTradingPair(ctx, baseAsset, quoteAsset)
		if err != nil {
			logger.Error("can not delist trading pair", "symbol", symbol, "err", err.Error())
			continue
		}

		if dexKeeper.ShouldPublishOrder() {
			pub.DelistTradingPairForPublish(ctx, dexKeeper, symbol)
		} else {
			dexKeeper.DelistTradingPair(ctx, symbol, nil)
		}
	}
}

func getSymbolsToDelist(ctx sdk.Context, govKeeper gov.Keeper, blockTime time.Time) []string {
	logger := bnclog.With("module", "dex")

	symbols := make([]string, 0)
	periodToSearch := getPeriodToSearch(ctx, govKeeper)

	govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, -1, true, func(proposal gov.Proposal) bool {
		// we do not need to search for all proposals
		if proposal.GetSubmitTime().Add(periodToSearch).Before(blockTime) {
			return true
		}

		if proposal.GetProposalType() == gov.ProposalTypeDelistTradingPair {
			var delistParam gov.DelistTradingPairParams
			err := json.Unmarshal([]byte(proposal.GetDescription()), &delistParam)
			if err != nil {
				logger.Error("illegal delist params in proposal", "params", proposal.GetDescription())
				return false
			}

			if delistParam.IsExecuted {
				return false
			}

			passedTime := proposal.GetVotingStartTime().Add(proposal.GetVotingPeriod())
			timeToDelist := passedTime.Add(DelayedDaysForDelist * 24 * time.Hour)
			if timeToDelist.Before(blockTime) {
				symbol := utils.Assets2TradingPair(strings.ToUpper(delistParam.BaseAssetSymbol), strings.ToUpper(delistParam.QuoteAssetSymbol))
				symbols = append(symbols, symbol)
				// update proposal delisted status
				delistParam.IsExecuted = true
				bz, err := json.Marshal(delistParam)
				if err != nil {
					logger.Error("marshal delist params error", "err", err.Error())
					return false
				}
				proposal.SetDescription(string(bz))
				govKeeper.SetProposal(ctx, proposal)
			}
		}
		return false
	})
	return symbols
}

func getPeriodToSearch(ctx sdk.Context, govKeeper gov.Keeper) time.Duration {
	depositParams := govKeeper.GetDepositParams(ctx)
	govMaxPeriod := depositParams.MaxDepositPeriod + gov.MaxVotingPeriod

	//add 2 days here for we search in breathe block, and the interval of breathe blocks is not exactly one day
	return govMaxPeriod + ((DelayedDaysForDelist + 2) * 24 * time.Hour)
}
