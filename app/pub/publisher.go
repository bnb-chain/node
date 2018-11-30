package pub

import (
	"fmt"
	"time"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
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
	cfg               *config.PublicationConfig
	ToPublishCh       chan BlockInfoToPublish
	ToRemoveOrderIdCh chan string // order ids to remove from keeper.OrderInfoForPublish
	IsLive            bool

	metrics *Metrics
)

type MarketDataPublisher interface {
	publish(msg AvroMsg, tpe msgType, height int64, timestamp int64)
	Stop()
}

func setup(
	logger tmlog.Logger,
	config *config.PublicationConfig,
	m *Metrics,
	publisher MarketDataPublisher) (err error) {
	Logger = logger.With("module", "pub")
	cfg = config
	metrics = m
	ToPublishCh = make(chan BlockInfoToPublish, config.PublicationChannelSize)
	if err = initAvroCodecs(); err != nil {
		Logger.Error("failed to initialize avro codec", "err", err)
		return err
	}

	go publish(publisher, logger, config, ToPublishCh)
	IsLive = true

	return nil
}

func publish(
	publisher MarketDataPublisher,
	Logger tmlog.Logger,
	cfg *config.PublicationConfig,
	ToPublishCh chan BlockInfoToPublish) {
	var lastPublishedTime time.Time
	for marketData := range ToPublishCh {
		Logger.Debug("publisher queue status", "size", len(ToPublishCh))
		metrics.PublicationQueueSize.Set(float64(len(ToPublishCh)))

		publishBlockTime := Timer(fmt.Sprintf("publish market data, height=%d", marketData.height), func() {
			// Implementation note: publication order are important here,
			// DEX query service team relies on the fact that we publish orders before trades so that
			// they can assign buyer/seller address into trade before persist into DB
			var opensToPublish []*order
			var canceledToPublish []*order
			var feeToPublish map[string]string
			if cfg.PublishOrderUpdates || cfg.PublishOrderBook {
				opensToPublish, canceledToPublish, feeToPublish = collectOrdersToPublish(
					marketData.tradesToPublish,
					marketData.orderChanges,
					marketData.orderChangesMap,
					marketData.feeHolder,
					marketData.timestamp)
				for _, o := range opensToPublish {
					if o.status == orderPkg.FullyFill {
						if ToRemoveOrderIdCh != nil {
							Logger.Debug(
								"going to delete fully filled order from order changes map",
								"orderId", o.orderId)
							ToRemoveOrderIdCh <- o.orderId
						}
					}
				}
				for _, o := range canceledToPublish {
					if ToRemoveOrderIdCh != nil {
						Logger.Debug(
							"going to delete order from order changes map",
							"orderId", o.orderId, "status", o.status)
						ToRemoveOrderIdCh <- o.orderId
					}
				}
			}

			// ToRemoveOrderIdCh would be only used in production code
			// will be nil in mock (pressure testing, local publisher) and test code
			if ToRemoveOrderIdCh != nil {
				close(ToRemoveOrderIdCh)
			}

			ordersToPublish := append(opensToPublish, canceledToPublish...)
			if cfg.PublishOrderUpdates {
				duration := Timer("publish all orders", func() {
					publishOrderUpdates(
						publisher,
						marketData.height,
						marketData.timestamp,
						ordersToPublish,
						marketData.tradesToPublish)
				})

				if metrics != nil {
					metrics.NumTrade.Set(float64(len(marketData.tradesToPublish)))
					metrics.NumOrder.Set(float64(len(ordersToPublish)))
					metrics.PublishTradeAndOrderTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishAccountBalance {
				duration := Timer("publish all changed accounts", func() {
					publishAccount(publisher, marketData.height, marketData.timestamp, marketData.accounts, feeToPublish)
				})

				if metrics != nil {
					metrics.NumAccounts.Set(float64(len(marketData.accounts)))
					metrics.PublishAccountTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishOrderBook {
				var changedPrices orderPkg.ChangedPriceLevelsMap
				duration := Timer("prepare order books to publish", func() {
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

				duration = Timer("publish changed order books", func() {
					publishOrderBookDelta(publisher, marketData.height, marketData.timestamp, changedPrices)
				})

				if metrics != nil {
					metrics.PublishOrderbookTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishBlockFee {
				duration := Timer("publish blockfee", func() {
					publishBlockFee(publisher, marketData.height, marketData.timestamp, marketData.blockFee)
				})

				if metrics != nil {
					metrics.PublishBlockfeeTimeMs.Set(float64(duration))
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

func publishOrderUpdates(publisher MarketDataPublisher, height int64, timestamp int64, os []*order, tradesToPublish []*Trade) {
	numOfOrders := len(os)
	numOfTrades := len(tradesToPublish)
	tradesAndOrdersMsg := tradesAndOrders{height: height, timestamp: timestamp, NumOfMsgs: numOfTrades + numOfOrders}
	if numOfOrders > 0 {
		tradesAndOrdersMsg.Orders = orders{numOfOrders, os}
	}
	if numOfTrades > 0 {
		tradesAndOrdersMsg.Trades = trades{numOfTrades, tradesToPublish}
	}

	publisher.publish(&tradesAndOrdersMsg, tradesAndOrdersTpe, height, timestamp)
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
	accountsMsg := accounts{height, numOfMsgs, accs}

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

func Timer(description string, op func()) (durationMs int64) {
	start := time.Now()
	op()
	durationMs = time.Since(start).Nanoseconds() / int64(time.Millisecond)
	Logger.Debug(description, "durationMs", durationMs)
	return durationMs
}
