package pub

import (
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/common/log"
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
)

type MarketDataPublisher interface {
	publish(msg AvroMsg, tpe msgType, height int64, timestamp int64)
	Stop()
}

func setup(config *config.PublicationConfig, publisher MarketDataPublisher) (err error) {
	Logger = log.With("module", "pub")
	cfg = config
	ToPublishCh = make(chan BlockInfoToPublish, config.PublicationChannelSize)
	if err = initAvroCodecs(); err != nil {
		Logger.Error("failed to initialize avro codec", "err", err)
		return err
	}

	go publish(publisher)
	IsLive = true

	return nil
}

func publish(publisher MarketDataPublisher) {
	for marketData := range ToPublishCh {
		// Implementation note: publication order are important here,
		// DEX query service team relies on the fact that we publish orders before trades so that
		// they can assign buyer/seller address into trade before persist into DB
		var opensToPublish []*order
		var canceledToPublish []*order
		if cfg.PublishOrderUpdates || cfg.PublishOrderBook {
			opensToPublish, canceledToPublish = collectOrdersToPublish(
				marketData.tradesToPublish,
				marketData.orderChanges,
				marketData.orderChangesMap,
				marketData.feeHolder,
				marketData.timestamp)
			for _, o := range opensToPublish {
				if o.status == orderPkg.FullyFill {
					Logger.Debug(
						"going to delete fully filled order from order changes map",
						"orderId", o.orderId)
					ToRemoveOrderIdCh <- o.orderId
				}
			}
			for _, o := range canceledToPublish {
				Logger.Debug(
					"going to delete order from order changes map",
					"orderId", o.orderId, "status", o.status)
				ToRemoveOrderIdCh <- o.orderId
			}
		}
		close(ToRemoveOrderIdCh)

		ordersToPublish := append(opensToPublish, canceledToPublish...)
		if cfg.PublishOrderUpdates {
			Logger.Debug("start to publish all orders")
			publishOrderUpdates(
				publisher,
				marketData.height,
				marketData.timestamp,
				ordersToPublish,
				marketData.tradesToPublish)
		}

		if cfg.PublishAccountBalance {
			Logger.Debug("start to publish all changed accounts")
			publishAccount(publisher, marketData.height, marketData.timestamp, marketData.accounts)
		}

		if cfg.PublishOrderBook {
			Logger.Debug("start to publish changed order books")
			changedPrices := filterChangedOrderBooksByOrders(ordersToPublish, marketData.latestPricesLevels)
			publishOrderBookDelta(publisher, marketData.height, marketData.timestamp, changedPrices)
		}

		if cfg.PublishBlockFee {
			Logger.Debug("start to publish blockfee")
			publishBlockFee(publisher, marketData.height, marketData.timestamp, marketData.blockFee)
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

func publishAccount(publisher MarketDataPublisher, height int64, timestamp int64, accountsToPublish map[string]Account) {
	numOfMsgs := len(accountsToPublish)

	idx := 0
	accs := make([]Account, numOfMsgs, numOfMsgs)
	for _, acc := range accountsToPublish {
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
