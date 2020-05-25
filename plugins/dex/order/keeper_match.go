package order

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/utils"
)

func (kp *DexKeeper) SelectSymbolsToMatch(height int64, matchAllSymbols bool) []string {
	symbolsToMatch := make([]string, 0, 256)
	for _, orderKeeper := range kp.OrderKeepers {
		if orderKeeper.supportUpgradeVersion() {
			symbolsToMatch = append(symbolsToMatch, orderKeeper.selectSymbolsToMatch(height, matchAllSymbols)...)
		}
	}
	return symbolsToMatch
}

func (kp *DexKeeper) MatchAndAllocateSymbols(ctx sdk.Context, postAlloTransHandler TransferHandler, matchAllSymbols bool) {
	kp.logger.Debug("Start Matching for all...", "height", ctx.BlockHeader().Height)
	blockHeader := ctx.BlockHeader()
	timestamp := blockHeader.Time.UnixNano()

	symbolsToMatch := kp.SelectSymbolsToMatch(blockHeader.Height, matchAllSymbols)
	kp.logger.Info("symbols to match", "symbols", symbolsToMatch)
	var tradeOuts []chan Transfer
	if len(symbolsToMatch) == 0 {
		kp.logger.Info("No order comes in for the block")
	} else {
		tradeOuts = kp.matchAndDistributeTrades(true, blockHeader.Height, timestamp)
	}

	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, postAlloTransHandler)
	fees.Pool.AddAndCommitFee("MATCH", totalFee)
	kp.ClearAfterMatch()
}

// please note if distributeTrade this method will work in async mode, otherwise in sync mode.
// Always run kp.SelectSymbolsToMatch(ctx.BlockHeader().Height, timestamp, matchAllSymbols) before matchAndDistributeTrades
func (kp *DexKeeper) matchAndDistributeTrades(distributeTrade bool, height, timestamp int64) []chan Transfer {
	concurrency := 1 << kp.poolSize
	tradeOuts := make([]chan Transfer, concurrency)

	if distributeTrade {
		ordNum := 0
		for i := range kp.OrderKeepers {
			ordNum += kp.OrderKeepers[i].getRoundOrdersNum()
		}

		for i := range tradeOuts {
			//assume every new order would have 2 trades and generate 4 transfer
			tradeOuts[i] = make(chan Transfer, ordNum*4/concurrency)
		}
	}

	symbolCh := make(chan string, concurrency)
	producer := func() {
		for i := range kp.OrderKeepers {
			kp.OrderKeepers[i].iterateRoundPairs(func(symbol string) {
				symbolCh <- symbol
			})
		}
		close(symbolCh)
	}
	matchWorker := func() {
		for symbol := range symbolCh {
			kp.matchAndDistributeTradesForSymbol(symbol, height, timestamp, distributeTrade, tradeOuts)
		}

	}

	if distributeTrade {
		utils.ConcurrentExecuteAsync(concurrency, producer, matchWorker, func() {
			for _, tradeOut := range tradeOuts {
				close(tradeOut)
			}
		})
	} else {
		utils.ConcurrentExecuteSync(concurrency, producer, matchWorker)
	}
	return tradeOuts
}

func (kp *DexKeeper) MatchSymbols(height, timestamp int64, matchAllSymbols bool) {
	symbolsToMatch := kp.SelectSymbolsToMatch(height, matchAllSymbols)
	kp.logger.Debug("symbols to match", "symbols", symbolsToMatch)

	if len(symbolsToMatch) == 0 {
		kp.logger.Info("No order comes in for the block")
	} else {
		kp.matchAndDistributeTrades(false, height, timestamp)
	}

	kp.ClearAfterMatch()
}

func (kp *DexKeeper) matchAndDistributeTradesForSymbol(symbol string, height, timestamp int64, distributeTrade bool,
	tradeOuts []chan Transfer) {
	engine := kp.engines[symbol]
	concurrency := len(tradeOuts)
	orderKeeper := kp.mustGetOrderKeeper(symbol)
	orders := orderKeeper.getAllOrdersForPair(symbol)
	// please note there is no logging in matching, expecting to see the order book details
	// from the exchange's order book stream.
	if engine.Match(height) {
		kp.logger.Debug("Match finish:", "symbol", symbol, "lastTradePrice", engine.LastTradePrice)
		for i := range engine.Trades {
			t := &engine.Trades[i]
			updateOrderMsg(orders[t.Bid], t.BuyCumQty, height, timestamp)
			updateOrderMsg(orders[t.Sid], t.SellCumQty, height, timestamp)
			if distributeTrade {
				t1, t2 := TransferFromTrade(t, symbol, orders)
				c := channelHash(t1.accAddress, concurrency)
				tradeOuts[c] <- t1
				c = channelHash(t2.accAddress, concurrency)
				tradeOuts[c] <- t2
			}
		}
		droppedIds := engine.DropFilledOrder() //delete from order books
		for _, id := range droppedIds {
			delete(orders, id) //delete from order cache
		}
		kp.logger.Debug("Drop filled orders", "total", droppedIds)
	} else {
		// FUTURE-TODO:
		// when Match() failed, have to unsolicited cancel all the new orders
		// in this block. Ideally the order IDs would be stored in the EndBlock response,
		// but this is not implemented yet, pending Tendermint to better handle EndBlock
		// for index service.
		kp.logger.Error("Fatal error occurred in matching, cancel all incoming new orders",
			"symbol", symbol)
		thisRoundIds := orderKeeper.getRoundOrdersForPair(symbol)
		for _, id := range thisRoundIds {
			msg := orders[id]
			delete(orders, id)
			if ord, err := engine.Book.RemoveOrder(id, msg.Side, msg.Price); err == nil {
				kp.logger.Info("Removed due to match failure", "ordID", msg.Id)
				if distributeTrade {
					c := channelHash(msg.Sender, concurrency)
					tradeOuts[c] <- TransferFromCanceled(ord, *msg, true)
				}
			} else {
				kp.logger.Error("Failed to remove order, may be fatal!", "orderID", id)
			}

			// let the order status publisher publish these abnormal
			// order status change outs.
			if kp.CollectOrderInfoForPublish {
				orderKeeper.appendOrderChangeSync(OrderChange{id, FailedMatching, "", nil})
			}
		}
		return // no need to handle IOC
	}
	var iocIDs []string
	iocIDs = orderKeeper.getRoundIOCOrdersForPair(symbol)
	for _, id := range iocIDs {
		if msg, ok := orders[id]; ok {
			delete(orders, id)
			if ord, err := engine.Book.RemoveOrder(id, msg.Side, msg.Price); err == nil {
				kp.logger.Debug("Removed unclosed IOC order", "ordID", msg.Id)
				if distributeTrade {
					c := channelHash(msg.Sender, concurrency)
					tradeOuts[c] <- TransferFromExpired(ord, *msg)
				}
			} else {
				kp.logger.Error("Failed to remove IOC order, may be fatal!", "orderID", id)
			}
		}
	}
}

// Run as postConsume procedure of async, no concurrent updates of orders map
func updateOrderMsg(order *OrderInfo, cumQty, height, timestamp int64) {
	order.CumQty = cumQty
	order.LastUpdatedHeight = height
	order.LastUpdatedTimestamp = timestamp
}
