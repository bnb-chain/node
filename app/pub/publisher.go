package pub

import (
	"fmt"
	"time"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/app/config"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
)

const (
	// TODO(#66): revisit the setting / whole thread model here,
	// do we need better way to make main thread less possibility to block
	TransferCollectionChannelSize = 4000
	ToRemoveOrderIdChannelSize    = 1000
	MaxOrderBookLevel             = 100
)

var (
	Logger            tmlog.Logger
	Cfg               *config.PublicationConfig
	ToPublishCh       chan BlockInfoToPublish
	ToRemoveOrderIdCh chan string // order ids to remove from keeper.OrderInfoForPublish
	IsLive            bool
)

type MarketDataPublisher interface {
	publish(msg AvroOrJsonMsg, tpe msgType, height int64, timestamp int64)
	Stop()
}

func Publish(
	publisher MarketDataPublisher,
	metrics *Metrics,
	Logger tmlog.Logger,
	cfg *config.PublicationConfig,
	ToPublishCh <-chan BlockInfoToPublish) {
	var lastPublishedTime time.Time
	for marketData := range ToPublishCh {
		Logger.Debug("publisher queue status", "size", len(ToPublishCh))
		if metrics != nil {
			metrics.PublicationQueueSize.Set(float64(len(ToPublishCh)))
		}

		publishBlockTime := Timer(Logger, fmt.Sprintf("publish market data, height=%d", marketData.height), func() {
			// Implementation note: publication order are important here,
			// DEX query service team relies on the fact that we publish orders before trades so that
			// they can assign buyer/seller address into trade before persist into DB
			var opensToPublish []*Order
			var closedToPublish []*Order
			var feeToPublish map[string]string
			if cfg.PublishOrderUpdates || cfg.PublishOrderBook {
				opensToPublish, closedToPublish, feeToPublish = collectOrdersToPublish(
					marketData.tradesToPublish,
					marketData.orderChanges,
					marketData.orderInfos,
					marketData.feeHolder,
					marketData.timestamp)
				for _, o := range closedToPublish {
					if ToRemoveOrderIdCh != nil {
						Logger.Debug(
							"going to delete order from order changes map",
							"orderId", o.OrderId, "status", o.Status)
						ToRemoveOrderIdCh <- o.OrderId
					}
				}
			}

			// ToRemoveOrderIdCh would be only used in production code
			// will be nil in mock (pressure testing, local publisher) and test code
			if ToRemoveOrderIdCh != nil {
				close(ToRemoveOrderIdCh)
			}

			ordersToPublish := append(opensToPublish, closedToPublish...)
			if cfg.PublishOrderUpdates {
				duration := Timer(Logger, "publish all orders", func() {
					publishExecutionResult(
						publisher,
						marketData.height,
						marketData.timestamp,
						ordersToPublish,
						marketData.tradesToPublish,
						marketData.proposalsToPublish,
						marketData.stakeUpdates)
				})

				if metrics != nil {
					metrics.NumTrade.Set(float64(len(marketData.tradesToPublish)))
					metrics.NumOrder.Set(float64(len(ordersToPublish)))
					metrics.PublishTradeAndOrderTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishAccountBalance {
				duration := Timer(Logger, "publish all changed accounts", func() {
					publishAccount(publisher, marketData.height, marketData.timestamp, marketData.accounts, feeToPublish)
				})

				if metrics != nil {
					metrics.NumAccounts.Set(float64(len(marketData.accounts)))
					metrics.PublishAccountTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishOrderBook {
				var changedPrices orderPkg.ChangedPriceLevelsMap
				duration := Timer(Logger, "prepare order books to publish", func() {
					changedPrices = filterChangedOrderBooksByOrders(ordersToPublish, marketData.latestPricesLevels)
				})
				if metrics != nil {
					numOfChangedPrices := 0
					for _, changedPrice := range changedPrices {
						numOfChangedPrices += len(changedPrice.Buys)
						numOfChangedPrices += len(changedPrice.Sells)
					}
					metrics.NumOrderBook.Set(float64(numOfChangedPrices))
					metrics.CollectOrderBookTimeMs.Set(float64(duration))
				}

				duration = Timer(Logger, "publish changed order books", func() {
					publishOrderBookDelta(publisher, marketData.height, marketData.timestamp, changedPrices)
				})

				if metrics != nil {
					metrics.PublishOrderbookTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishBlockFee {
				duration := Timer(Logger, "publish blockfee", func() {
					publishBlockFee(publisher, marketData.height, marketData.timestamp, marketData.blockFee)
				})

				if metrics != nil {
					metrics.PublishBlockfeeTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishTransfer {
				duration := Timer(Logger, "publish transfers", func() {
					publishTransfers(publisher, marketData.height, marketData.timestamp, marketData.transfers)
				})
				if metrics != nil {
					metrics.NumTransfers.Set(float64(len(marketData.transfers.Transfers)))
					metrics.PublishTransfersTimeMs.Set(float64(duration))
				}
			}

			if metrics != nil {
				metrics.PublicationHeight.Set(float64(marketData.height))
				blockInterval := time.Since(lastPublishedTime)
				lastPublishedTime = time.Now()
				metrics.PublicationBlockIntervalMs.Set(float64(blockInterval.Nanoseconds() / int64(time.Millisecond)))
			}
		})

		if metrics != nil {
			metrics.PublishBlockTimeMs.Set(float64(publishBlockTime))
		}
	}
}

func Stop(publisher MarketDataPublisher) {
	if IsLive == false {
		Logger.Error("publication module has already been stopped")
		return
	}

	IsLive = false

	close(ToPublishCh)
	if ToRemoveOrderIdCh != nil {
		close(ToRemoveOrderIdCh)
	}

	publisher.Stop()
}

func publishExecutionResult(publisher MarketDataPublisher, height int64, timestamp int64, os []*Order, tradesToPublish []*Trade, proposalsToPublish *Proposals, stakeUpdates *StakeUpdates) {
	numOfOrders := len(os)
	numOfTrades := len(tradesToPublish)
	numOfProposals := proposalsToPublish.NumOfMsgs
	numOfStakeUpdatedAccounts := stakeUpdates.NumOfMsgs
	executionResultsMsg := ExecutionResults{Height: height, Timestamp: timestamp, NumOfMsgs: numOfTrades + numOfOrders + numOfProposals + numOfStakeUpdatedAccounts}
	if numOfOrders > 0 {
		executionResultsMsg.Orders = Orders{numOfOrders, os}
	}
	if numOfTrades > 0 {
		executionResultsMsg.Trades = trades{numOfTrades, tradesToPublish}
	}
	if numOfProposals > 0 {
		executionResultsMsg.Proposals = *proposalsToPublish
	}
	if numOfStakeUpdatedAccounts > 0 {
		executionResultsMsg.StakeUpdates = *stakeUpdates
	}

	publisher.publish(&executionResultsMsg, executionResultTpe, height, timestamp)
}

func publishAccount(publisher MarketDataPublisher, height int64, timestamp int64, accountsToPublish map[string]Account, feeToPublish map[string]string) {
	numOfMsgs := len(accountsToPublish)

	idx := 0
	accs := make([]Account, numOfMsgs, numOfMsgs)
	for _, acc := range accountsToPublish {
		if fee, ok := feeToPublish[acc.Owner]; ok {
			acc.Fee = fee
		}
		accs[idx] = acc
		idx++
	}
	accountsMsg := Accounts{height, numOfMsgs, accs}

	publisher.publish(&accountsMsg, accountsTpe, height, timestamp)
}

func publishOrderBookDelta(publisher MarketDataPublisher, height int64, timestamp int64, changedPriceLevels orderPkg.ChangedPriceLevelsMap) {
	var deltas []OrderBookDelta
	for pair, pls := range changedPriceLevels {
		buys := make([]PriceLevel, len(pls.Buys), len(pls.Buys))
		sells := make([]PriceLevel, len(pls.Sells), len(pls.Sells))
		idx := 0
		for price, qty := range pls.Buys {
			buys[idx] = PriceLevel{price, qty}
			idx++
		}
		idx = 0
		for price, qty := range pls.Sells {
			sells[idx] = PriceLevel{price, qty}
			idx++
		}
		deltas = append(deltas, OrderBookDelta{pair, buys, sells})
	}

	books := Books{height, timestamp, len(deltas), deltas}

	publisher.publish(&books, booksTpe, height, timestamp)
}

func publishBlockFee(publisher MarketDataPublisher, height, timestamp int64, blockFee BlockFee) {
	publisher.publish(blockFee, blockFeeTpe, height, timestamp)
}

func publishTransfers(publisher MarketDataPublisher, height, timestamp int64, transfers *Transfers) {
	if transfers != nil {
		publisher.publish(transfers, transferType, height, timestamp)
	}
}

func Timer(logger tmlog.Logger, description string, op func()) (durationMs int64) {
	start := time.Now()
	op()
	durationMs = time.Since(start).Nanoseconds() / int64(time.Millisecond)
	logger.Debug(description, "durationMs", durationMs)
	return durationMs
}
