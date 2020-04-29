package order

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/utils"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
)

func MatchAndAllocateSymbols(dexKeeper *Keeper, dexMiniKeeper *MiniKeeper, ctx sdk.Context, postAlloTransHandler TransferHandler, matchAllSymbols bool, logger tmlog.Logger) {
	logger.Debug("Start Matching for all...", "height", ctx.BlockHeader().Height, "symbolNum", len(dexKeeper.roundOrders))
	timestamp := ctx.BlockHeader().Time.UnixNano()

	symbolsToMatch := dexKeeper.symbolSelector.SelectSymbolsToMatch(dexKeeper.roundOrders, ctx.BlockHeader().Height, timestamp, matchAllSymbols)
	symbolsToMatch = append(symbolsToMatch, dexKeeper.symbolSelector.SelectSymbolsToMatch(dexKeeper.roundOrders, ctx.BlockHeader().Height, timestamp, matchAllSymbols)...)
	tradeOuts := matchAndDistributeTrades(dexKeeper, dexMiniKeeper, true, ctx.BlockHeader().Height, timestamp, symbolsToMatch, logger)
	if tradeOuts == nil {
		logger.Info("No order comes in for the block")
	}
	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, postAlloTransHandler)
	fees.Pool.AddAndCommitFee("MATCH", totalFee)
	clearAfterMatchBEP2(dexKeeper)
	clearAfterMatchMini(dexMiniKeeper)
}

func clearAfterMatchBEP2(kp *Keeper) {
	kp.logger.Debug("clearAfterMatchBEP2...")
	kp.roundOrders = make(map[string][]string, 256)
	kp.roundIOCOrders = make(map[string][]string, 256)
}

func clearAfterMatchMini(kp *MiniKeeper) {
	kp.logger.Debug("clearAfterMatchMini...")
	for _, symbol := range *kp.symbolSelector.GetRoundMatchSymbol() {
		delete(kp.roundOrders, symbol)
		delete(kp.roundIOCOrders, symbol)
	}
	emptyRoundMatchSymbols := make([]string, 0, 256)
	kp.symbolSelector.SetRoundMatchSymbol(emptyRoundMatchSymbols)
}

// please note if distributeTrade this method will work in async mode, otherwise in sync mode.
func matchAndDistributeTrades(dexKeeper *Keeper, dexMiniKeeper *MiniKeeper, distributeTrade bool, height, timestamp int64, symbolsToMatch []string, logger tmlog.Logger) []chan Transfer {
	if len(symbolsToMatch) == 0 {
		logger.Info("No symbols to match in the block")
		return nil
	}
	concurrency := 1 << dexUtils.MaxOf(dexKeeper.poolSize, dexMiniKeeper.poolSize)
	tradeOuts := make([]chan Transfer, concurrency)

	if distributeTrade {
		ordNum := 0
		for _, symbol := range symbolsToMatch {
			if dexUtils.IsMiniTokenTradingPair(symbol) {
				ordNum += len(dexMiniKeeper.roundOrders[symbol])
			} else {
				ordNum += len(dexKeeper.roundOrders[symbol])
			}
		}
		for i := range tradeOuts {
			//assume every new order would have 2 trades and generate 4 transfer
			tradeOuts[i] = make(chan Transfer, ordNum*4/concurrency)
		}
	}

	symbolCh := make(chan string, concurrency)
	producer := func() {
		for _, symbol := range symbolsToMatch {
			symbolCh <- symbol
		}
		close(symbolCh)
	}
	matchWorker := func() {
		for symbol := range symbolCh {
			if dexUtils.IsMiniTokenTradingPair(symbol) {
				dexMiniKeeper.matchAndDistributeTradesForSymbol(symbol, height, timestamp, dexMiniKeeper.allOrders[symbol], distributeTrade, tradeOuts)
			} else {
				dexKeeper.matchAndDistributeTradesForSymbol(symbol, height, timestamp, dexKeeper.allOrders[symbol], distributeTrade, tradeOuts)
			}
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

func (kp *Keeper) MatchSymbols(height, timestamp int64) {
	symbolsToMatch := kp.symbolSelector.SelectSymbolsToMatch(kp.roundOrders, height, timestamp, false)
	kp.logger.Debug("symbols to match", "symbols", symbolsToMatch)
	tradeOuts := kp.matchAndDistributeTrades(false, height, timestamp, symbolsToMatch) //only match
	if tradeOuts == nil {
		kp.logger.Info("No order comes in for the block")
	}
	clearAfterMatchBEP2(dexKeeper)
	clearAfterMatchMini(dexMiniKeeper)
}

// MatchAndAllocateSymbols() is concurrently matching and allocating across
// all the symbols' order books, among all the clients
// Return whether match has been done in this height
//func (kp *Keeper) MatchAndAllocateSymbols(ctx sdk.Context, postAlloTransHandler TransferHandler, matchAllSymbols bool) {
//	kp.logger.Debug("Start Matching for all...", "height", ctx.BlockHeader().Height, "symbolNum", len(kp.roundOrders))
//	timestamp := ctx.BlockHeader().Time.UnixNano()
//
//	symbolsToMatch := kp.symbolSelector.SelectSymbolsToMatch(kp.roundOrders, ctx.BlockHeader().Height, timestamp, matchAllSymbols)
//	tradeOuts := kp.matchAndDistributeTrades(true, ctx.BlockHeader().Height, timestamp, symbolsToMatch)
//	if tradeOuts == nil {
//		kp.logger.Info("No order comes in for the block")
//	}
//	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, postAlloTransHandler)
//	fees.Pool.AddAndCommitFee("MATCH", totalFee)
//	kp.clearAfterMatch(kp)
//}

//// please note if distributeTrade this method will work in async mode, otherwise in sync mode.
//func (kp *Keeper) matchAndDistributeTrades(distributeTrade bool, height, timestamp int64, symbolsToMatch []string) []chan Transfer {
//	if len(symbolsToMatch) == 0 {
//		kp.logger.Info("No symbols to match in the block")
//		return nil
//	}
//	concurrency := 1 << kp.poolSize
//	tradeOuts := make([]chan Transfer, concurrency)
//
//	if distributeTrade {
//		ordNum := 0
//		for _, symbol := range symbolsToMatch {
//			ordNum += len(kp.roundOrders[symbol])
//		}
//		for i := range tradeOuts {
//			//assume every new order would have 2 trades and generate 4 transfer
//			tradeOuts[i] = make(chan Transfer, ordNum*4/concurrency)
//		}
//	}
//
//	symbolCh := make(chan string, concurrency)
//	producer := func() {
//		for _, symbol := range symbolsToMatch {
//			symbolCh <- symbol
//		}
//		close(symbolCh)
//	}
//	matchWorker := func() {
//		i := 0
//		for symbol := range symbolCh {
//			i++
//			kp.matchAndDistributeTradesForSymbol(symbol, height, timestamp, kp.allOrders[symbol], distributeTrade, tradeOuts)
//		}
//	}
//
//	if distributeTrade {
//		utils.ConcurrentExecuteAsync(concurrency, producer, matchWorker, func() {
//			for _, tradeOut := range tradeOuts {
//				close(tradeOut)
//			}
//		})
//	} else {
//		utils.ConcurrentExecuteSync(concurrency, producer, matchWorker)
//	}
//	return tradeOuts
//}

func (kp *Keeper) matchAndDistributeTradesForSymbol(symbol string, height, timestamp int64, orders map[string]*OrderInfo,
	distributeTrade bool, tradeOuts []chan Transfer) {
	engine := kp.engines[symbol]
	concurrency := len(tradeOuts)
	// please note there is no logging in matching, expecting to see the order book details
	// from the exchange's order book stream.
	if engine.Match(height) {
		kp.logger.Debug("Match finish:", "symbol", symbol, "lastTradePrice", engine.LastTradePrice)
		for i := range engine.Trades {
			t := &engine.Trades[i]
			updateOrderMsg(orders[t.Bid], t.BuyCumQty, height, timestamp)
			updateOrderMsg(orders[t.Sid], t.SellCumQty, height, timestamp)
			if distributeTrade {
				t1, t2 := TransferFromTrade(t, symbol, kp.allOrders[symbol])
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
		thisRoundIds := kp.roundOrders[symbol]
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
				kp.OrderChangesMtx.Lock()
				kp.OrderChanges = append(kp.OrderChanges, OrderChange{id, FailedMatching, "", nil})
				kp.OrderChangesMtx.Unlock()
			}
		}
		return // no need to handle IOC
	}
	var iocIDs []string
	iocIDs = kp.roundIOCOrders[symbol]
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