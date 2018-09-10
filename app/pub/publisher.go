package pub

import (
	"fmt"
	"github.com/Shopify/sarama"
	"strconv"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

const (
	// TODO(#66): revisit the setting / whole thread model here, do we need better way to make main thread less possibility to block
	PublicationBufferSize = 10000
	// TODO(#66): map from flag --> topic --> broker cluster should be saved in configure file
	topic = "test"
)

var (
	// TODO(#66): map from flag --> topic --> broker cluster should be saved in configure file
	//brokers = []string{"192.168.15.38:9092"}
	brokers = []string{"127.0.0.1:9092"}
)

type MarketDataPublisher struct {
	Logger           log.Logger
	ToPublishChannel chan BlockInfoToPublish
	IsLive           bool // TODO(#66): thread safty: is EndBlocker and Init are call in same thread?

	producer sarama.SyncProducer
}

func (publisher *MarketDataPublisher) Init() (err error) {
	if publisher.producer, err = newProducer(); err != nil {
		publisher.Logger.Error("failed to create new kafka producer", "err", err)
		return err
	}
	initAvroCodecs(publisher.Logger)
	go publisher.publishMarketData()
	publisher.IsLive = true
	return nil
}

func (publisher *MarketDataPublisher) Stop() {
	publisher.Logger.Info("start to stop MarketDataPublisher")
	publisher.IsLive = false
	close(publisher.ToPublishChannel)
	// nil check because this method would be called when we failed to create producer
	if publisher.producer != nil {
		publisher.producer.Close()
	}
	publisher.Logger.Info("finished stop MarketDataPublisher")
}

func (publisher *MarketDataPublisher) publishMarketData() {
	for marketData := range publisher.ToPublishChannel {
		totalMsgs := 0

		// Implementation note: publication order are important here, DEX query service team relies on the fact that we publish orders before trades so that they can assign buyer/seller address into trade before persist into DB

		publisher.Logger.Info("start to publish all orders")
		ordersToPublish := publisher.collectFilledOrdersFromTrade(&marketData.tradesForAllPairs, marketData.orderChanges, marketData.orderChangesMap)
		totalMsgs += publisher.publishOrderData(marketData.height, marketData.timestamp, ordersToPublish)

		publisher.Logger.Info("start to publish all trade data")
		totalMsgs += publisher.publishTradeData(marketData.height, marketData.timestamp, &marketData.tradesForAllPairs)

		publisher.Logger.Info("start to publish changed order books")
		changedPrices := publisher.collectChangedOrderBooksFromOrders(&ordersToPublish, marketData.latestPricesLevels)
		totalMsgs += publisher.publishOrderBookData(marketData.height, marketData.timestamp, changedPrices)

		publisher.Logger.Info("start to publish block committed msg")
		publisher.publishBlockCommitted(marketData.height, marketData.timestamp, totalMsgs)
	}
}

func (publisher *MarketDataPublisher) publishTradeData(height int64, timestamp int64, tradesForAllPairs *map[string][]matcheng.Trade) int {
	if len(*tradesForAllPairs) == 0 {
		return 0
	}
	var idx = 0
	var tradesToPublish []trade
	for pair, trades := range *tradesForAllPairs {
		for _, t := range trades {
			// TODO(https://github.com/BiJie/BinanceChain/issues/93): correct fees
			tradesToPublish = append(tradesToPublish, trade{fmt.Sprintf("%d-%d", height, idx), pair, t.LastPx, t.LastQty, t.SId, t.BId, 0, 0})
			idx += 1
		}
	}
	tradesMsg := trades{height, idx, tradesToPublish}
	if msg, err := marshal(&tradesMsg, tradesTpe); err == nil {
		kafkaMsg := publisher.prepareMessage(topic, strconv.FormatInt(height, 10), timestamp, tradesTpe, msg)
		if partition, offset, err := publisher.producer.SendMessage(kafkaMsg); err == nil {
			publisher.Logger.Info(fmt.Sprintf("published trades: %s, at offset: %d (of partition: %d)", tradesMsg.String(), offset, partition))
		} else {
			// TODO(#66): pattern match against error types
			publisher.Logger.Error(fmt.Sprintf("failed to publish trades: %s", tradesMsg.String()), "err", err)
		}
	} else {
		publisher.Logger.Error(fmt.Sprintf("failed to publish trades: %s", tradesMsg.String()), "err", err)
	}
	return idx
}

func (publisher *MarketDataPublisher) publishOrderData(height int64, timestamp int64, os []order) int {
	if len(os) == 0 {
		return 0
	}
	ordersMsg := orders{height, len(os), os}
	if msg, err := marshal(&ordersMsg, ordersTpe); err == nil {
		kafkaMsg := publisher.prepareMessage(topic, strconv.FormatInt(height, 10), timestamp, ordersTpe, msg)
		if partition, offset, err := publisher.producer.SendMessage(kafkaMsg); err == nil {
			publisher.Logger.Info(fmt.Sprintf("published orders: %s, at offset: %d (of partition: %d)", ordersMsg.String(), offset, partition))
		} else {
			// TODO(#66): pattern match against error types
			publisher.Logger.Error(fmt.Sprintf("failed to publish orders: %s", ordersMsg.String()), "err", err)
		}
	} else {
		publisher.Logger.Error(fmt.Sprintf("failed to publish orders: %s", ordersMsg.String()), "err", err)
	}
	return len(os)
}

// we collect OrderPart here to make matcheng module independent
func (publisher *MarketDataPublisher) collectFilledOrdersFromTrade(trades *map[string][]matcheng.Trade, orderChanges orderPkg.OrderChanges, orderChangesMap orderPkg.OrderChangesMap) (ordersToPublish []order) {
	ordersToPublish = make([]order, len(orderChanges))
	idx := 0

	// collect orders (new, cancel, ioc-no-fill, expire) from orderChanges
	for _, o := range orderChanges {
		var executedQty int64
		switch o.Tpe {
		case orderPkg.Canceled:
			executedQty = o.LeavesQty
		case orderPkg.IocNoFill:
			executedQty = o.LeavesQty
		case orderPkg.Expired:
			executedQty = o.LeavesQty
		default:
			executedQty = o.OrderMsg.Quantity
		}
		// TODO: fill up cumQuoteAssetQty
		ordersToPublish[idx] = order{o.OrderMsg.Symbol, o.Tpe.String(), o.OrderMsg.Id, "-1", o.OrderMsg.Sender.String(), orderPkg.IToSide(o.OrderMsg.Side), "LIMIT", o.OrderMsg.Price, o.OrderMsg.Quantity, o.OrderMsg.Price, executedQty, o.CumQty, 0, 0, "", 0, 0, orderPkg.IToTimeInForce(o.OrderMsg.TimeInForce), "NEW"}
		idx += 1
	}

	// collect orders from trades
	for _, tradesPerPair := range *trades {
		for _, t := range tradesPerPair {

			if o, ok := orderChangesMap[t.BId]; ok {
				var status orderPkg.ChangeType
				if t.BuyCumQty == o.OrderMsg.Quantity {
					status = orderPkg.FullyFill
				} else {
					status = orderPkg.PartialFill
				}
				ordersToPublish = append(ordersToPublish, order{o.OrderMsg.Symbol, status.String(), o.OrderMsg.Id, "-1", o.OrderMsg.Sender.String(), orderPkg.IToSide(o.OrderMsg.Side), "LIMIT", o.OrderMsg.Price, o.OrderMsg.Quantity, t.LastPx, t.LastQty, t.BuyCumQty, 0, 0, "", 0, 0, orderPkg.IToTimeInForce(o.OrderMsg.TimeInForce), "NEW"})
			} else {
				publisher.Logger.Error("failed to resolve order information for id: from orderChangesMap" + t.BId)
			}

			if o, ok := orderChangesMap[t.SId]; ok {
				// TODO: revisit how to tell a sell order is partial fill or fully filled
				var status orderPkg.ChangeType
				if t.BuyCumQty == o.OrderMsg.Quantity {
					status = orderPkg.FullyFill
				} else {
					status = orderPkg.PartialFill
				}
				ordersToPublish = append(ordersToPublish, order{o.OrderMsg.Symbol, status.String(), o.OrderMsg.Id, "-1", o.OrderMsg.Sender.String(), orderPkg.IToSide(o.OrderMsg.Side), "LIMIT", o.OrderMsg.Price, o.OrderMsg.Quantity, t.LastPx, t.LastQty, t.BuyCumQty, 0, 0, "", 0, 0, orderPkg.IToTimeInForce(o.OrderMsg.TimeInForce), "NEW"})
			} else {
				publisher.Logger.Error("failed to resolve order information for id: from orderChangesMap" + t.SId)
			}
		}
	}
	return ordersToPublish
}

// collect all changed books according to published order status
func (publisher *MarketDataPublisher) collectChangedOrderBooksFromOrders(ordersToPublish *[]order, latestPriceLevels orderPkg.ChangedPriceLevels) orderPkg.ChangedPriceLevels {
	var res = make(orderPkg.ChangedPriceLevels)
	var buySideStr = orderPkg.IToSide(orderPkg.Side.BUY)
	var sellSideStr = orderPkg.IToSide(orderPkg.Side.SELL)
	for _, o := range *ordersToPublish {
		if _, ok := latestPriceLevels[o.symbol]; !ok {
			continue
		}
		if _, ok := res[o.symbol]; !ok {
			res[o.symbol] = orderPkg.ChangedPriceLevelsPerSymbol{make(map[int64]int64), make(map[int64]int64)}
		}

		switch o.side {
		case buySideStr:
			// TODO: code clean up - here we rely on special implementation that for orders that not generated from trade (like New, Cancel) the lastExecutedPrice is original price (rather than 0)
			if qty, ok := latestPriceLevels[o.symbol].Buys[o.lastExecutedPrice]; ok {
				res[o.symbol].Buys[o.lastExecutedPrice] = qty
			} else {
				res[o.symbol].Buys[o.lastExecutedPrice] = 0
			}
		case sellSideStr:
			if qty, ok := latestPriceLevels[o.symbol].Buys[o.lastExecutedPrice]; ok {
				res[o.symbol].Sells[o.lastExecutedPrice] = qty
			} else {
				res[o.symbol].Sells[o.lastExecutedPrice] = 0
			}
		}
	}
	return res
}

func (publisher *MarketDataPublisher) publishOrderBookData(height int64, timestamp int64, changedPriceLevels orderPkg.ChangedPriceLevels) int {
	var deltas []orderBookDelta
	for pair, pls := range changedPriceLevels {
		buys := make([]priceLevel, len(pls.Buys), len(pls.Buys))
		sells := make([]priceLevel, len(pls.Sells), len(pls.Sells))
		idx := 0
		for price, qty := range pls.Buys {
			buys[idx] = priceLevel{price, qty}
			idx++
		}
		idx = 0
		for price, qty := range pls.Sells {
			sells[idx] = priceLevel{price, qty}
			idx++
		}
		deltas = append(deltas, orderBookDelta{pair, buys, sells})
	}

	if len(deltas) != 0 {
		books := books{height, len(deltas), deltas}
		if msg, err := marshal(&books, booksTpe); err == nil {
			kafkaMsg := publisher.prepareMessage(topic, strconv.FormatInt(height, 10), timestamp, booksTpe, msg)
			if partition, offset, err := publisher.producer.SendMessage(kafkaMsg); err == nil {
				publisher.Logger.Info(fmt.Sprintf("published books: %s, at offset: %d (of partition: %d)", books.String(), offset, partition))
			} else {
				// TODO(#66): pattern match against error types
				publisher.Logger.Error(fmt.Sprintf("failed to publish books: %s", books.String()), "err", err)
			}
		} else {
			publisher.Logger.Error(fmt.Sprintf("failed to publish books: %s", books.String()), "err", err)
		}
	}
	return len(deltas)
}

func (publisher *MarketDataPublisher) publishBlockCommitted(height int64, timestamp int64, numOfMsgs int) {
	blockCommited := blockCommitted{height, "commited", timestamp, numOfMsgs}
	if msg, err := marshal(&blockCommited, blockCommitedTpe); err == nil {
		kafkaMsg := publisher.prepareMessage(topic, strconv.FormatInt(height, 10), timestamp, blockCommitedTpe, msg)
		if partition, offset, err := publisher.producer.SendMessage(kafkaMsg); err == nil {
			publisher.Logger.Info(fmt.Sprintf("published block commit: %d, at offset: %d (of partition: %d)", height, offset, partition))
		} else {
			// TODO(#66): pattern match against error types
			publisher.Logger.Error(fmt.Sprintf("failed to publish block commit: %d", height), "err", err)
		}
	} else {
		publisher.Logger.Error(fmt.Sprintf("failed to publish block commit: %d", height), "err", err)
	}
}

func newProducer() (sarama.SyncProducer, error) {
	// TODO: revisit configurations here
	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 20
	// This MIGHT be kafka java client's equivalent max.in.flight.requests.per.connection to make sure messages won't out-of-order
	// Refer: https://github.com/Shopify/sarama/issues/718
	config.Net.MaxOpenRequests = 1
	producer, err := sarama.NewSyncProducer(brokers, config)

	return producer, err
}

func (publisher *MarketDataPublisher) prepareMessage(topic string, msgId string, timeStamp int64, msgTpe msgType, message []byte) *sarama.ProducerMessage {
	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Partition: -1,
		Key:       sarama.StringEncoder(fmt.Sprintf("%s_%d_%s", msgId, timeStamp, msgTpe.String())),
		Value:     sarama.ByteEncoder(message),
	}

	return msg
}
