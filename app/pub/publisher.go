package pub

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	"github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/common/log"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

const (
	// TODO(#66): revisit the setting / whole thread model here,
	// do we need better way to make main thread less possibility to block
	PublicationChannelSize     = 10000
	FeeCollectionChannelSize   = 4000
	ToRemoveOrderIdChannelSize = 1000
	MaxOrderBookLevel          = 20
)

var (
	Logger tmlog.Logger
)

type MarketDataPublisher struct {
	ToPublishCh       chan BlockInfoToPublish
	ToRemoveOrderIdCh chan string // order ids to remove from keeper.OrderInfoForPublish
	IsLive            bool        // TODO(#66): thread safty: is EndBlocker and Init are call in same thread?

	config    *config.PublicationConfig
	producers map[string]sarama.SyncProducer // topic -> producer
}

func NewMarketDataPublisher(config *config.PublicationConfig) (publisher *MarketDataPublisher) {
	publisher = &MarketDataPublisher{
		ToPublishCh: make(chan BlockInfoToPublish, PublicationChannelSize),
		config:      config,
		producers:   make(map[string]sarama.SyncProducer),
	}
	if err := publisher.init(log.With("module", "pub")); err != nil {
		publisher.Stop()
		log.Error("Cannot start up market data kafka publisher", "err", err)
		panic(err)
	}
	return publisher
}

func (publisher *MarketDataPublisher) init(logger tmlog.Logger) (err error) {
	sarama.Logger = saramaLogger{}
	Logger = logger

	if config, err := publisher.newProducers(); err != nil {
		Logger.Error("failed to create new kafka producer", "err", err)
		return err
	} else {
		// we have to use the same prometheus registerer with tendermint
		// so that we can share same host:port for prometheus daemon
		prometheusRegistry := prometheus.DefaultRegisterer
		metricsRegistry := config.MetricRegistry
		pClient := prometheusmetrics.NewPrometheusProvider(
			metricsRegistry,
			"",
			"publication",
			prometheusRegistry,
			1*time.Second)
		go pClient.UpdatePrometheusMetrics()
	}

	if err = initAvroCodecs(); err != nil {
		Logger.Error("failed to initialize avro codec", "err", err)
		return err
	}

	go publisher.publish()
	publisher.IsLive = true

	return nil
}

func (publisher *MarketDataPublisher) Stop() {
	Logger.Debug("start to stop MarketDataPublisher")
	publisher.IsLive = false

	close(publisher.ToPublishCh)
	close(publisher.ToRemoveOrderIdCh)

	for topic, producer := range publisher.producers {
		// nil check because this method would be called when we failed to create producer
		if producer != nil {
			if err := producer.Close(); err != nil {
				Logger.Error("failed to stop producer for topic", "topic", topic, "err", err)
			}
		}
	}
	Logger.Debug("finished stop MarketDataPublisher")
}

func (publisher *MarketDataPublisher) publish() {
	for marketData := range publisher.ToPublishCh {
		// Implementation note: publication order are important here,
		// DEX query service team relies on the fact that we publish orders before trades so that
		// they can assign buyer/seller address into trade before persist into DB
		var opensToPublish []order
		var canceledToPublish []order
		if publisher.config.PublishOrderUpdates || publisher.config.PublishOrderBook {
			opensToPublish, canceledToPublish = publisher.collectExecutedOrdersToPublish(
				&marketData.tradesToPublish,
				marketData.orderChanges,
				marketData.orderChangesMap,
				marketData.timestamp)
			for _, o := range opensToPublish {
				if o.status == orderPkg.FullyFill {
					Logger.Debug(
						"going to delete fully filled order from order changes map",
						"orderId", o.orderId)
					publisher.ToRemoveOrderIdCh <- o.orderId
				}
			}
			for _, o := range canceledToPublish {
				Logger.Debug(
					"going to delete order from order changes map",
					"orderId", o.orderId, "status", o.status)
				publisher.ToRemoveOrderIdCh <- o.orderId
			}
		}
		close(publisher.ToRemoveOrderIdCh)

		ordersToPublish := append(opensToPublish, canceledToPublish...)
		if publisher.config.PublishOrderUpdates {
			Logger.Debug("start to publish all orders")
			publisher.publishOrderUpdates(
				marketData.height,
				marketData.timestamp,
				ordersToPublish,
				marketData.tradesToPublish)
		}

		if publisher.config.PublishAccountBalance {
			Logger.Debug("start to publish all changed accounts")
			publisher.publishAccount(marketData.height, marketData.timestamp, marketData.accounts)
		}

		if publisher.config.PublishOrderBook {
			Logger.Debug("start to publish changed order books")
			changedPrices := publisher.filterChangedOrderBooksByOrders(ordersToPublish, marketData.latestPricesLevels)
			publisher.publishOrderBookData(marketData.height, marketData.timestamp, changedPrices)
		}
	}
}

// we collect OrderPart here to make matcheng module independent
func (publisher *MarketDataPublisher) collectExecutedOrdersToPublish(
	trades *[]Trade,
	orderChanges orderPkg.OrderChanges,
	orderChangesMap orderPkg.OrderInfoForPublish,
	timestamp int64) (opensToPublish []order, canceledToPublish []order) {
	opensToPublish = make([]order, 0)
	canceledToPublish = make([]order, 0)

	// collect orders (new, cancel, ioc-no-fill, expire) from orderChanges
	for _, o := range orderChanges {
		orderInfo := orderChangesMap[o.Id]
		orderToPublish := order{
			orderInfo.Symbol,
			o.Tpe,
			o.Id,
			"",
			orderInfo.Sender.String(),
			orderInfo.Side,
			orderPkg.OrderType.LIMIT,
			orderInfo.Price,
			orderInfo.Quantity,
			0,
			0,
			orderInfo.CumQty,
			o.Fee,
			o.FeeAsset,
			orderInfo.CreatedTimestamp,
			timestamp,
			orderInfo.TimeInForce,
			orderPkg.NEW,
			orderInfo.TxHash,
		}
		if o.Tpe == orderPkg.Ack {
			opensToPublish = append(opensToPublish, orderToPublish)
		} else {
			canceledToPublish = append(canceledToPublish, orderToPublish)
		}
	}

	// collect orders from trades
	for _, t := range *trades {
		if o, exists := orderChangesMap[t.Bid]; exists {
			opensToPublish = append(opensToPublish, publisher.tradeToOrder(t, o, timestamp))
		} else {
			Logger.Error("failed to resolve order information from orderChangesMap", "orderId", t.Bid)
		}

		if o, exists := orderChangesMap[t.Sid]; exists {
			opensToPublish = append(opensToPublish, publisher.tradeToOrder(t, o, timestamp))
		} else {
			Logger.Error("failed to resolve order information from orderChangesMap", "orderId", t.Sid)
		}
	}

	return opensToPublish, canceledToPublish
}

func (publisher *MarketDataPublisher) tradeToOrder(
	t Trade,
	o *orderPkg.OrderInfo,
	timestamp int64) order {

	var status orderPkg.ChangeType
	if o.CumQty == o.Quantity {
		status = orderPkg.FullyFill
	} else {
		status = orderPkg.PartialFill
	}
	var fee int64
	var feeAsset string
	if o.Side == orderPkg.Side.BUY {
		fee = t.Bfee
		feeAsset = t.BfeeAsset
	} else {
		fee = t.Sfee
		feeAsset = t.SfeeAsset
	}
	res := order{
		o.Symbol,
		status,
		o.Id,
		t.Id,
		o.Sender.String(),
		o.Side,
		orderPkg.OrderType.LIMIT,
		o.Price,
		o.Quantity,
		t.Price,
		t.Qty,
		o.CumQty,
		fee,
		feeAsset,
		o.CreatedTimestamp,
		timestamp,
		o.TimeInForce,
		orderPkg.NEW,
		o.TxHash,
	}
	return res
}

func (publisher *MarketDataPublisher) publishOrderUpdates(height int64, timestamp int64, os []order, tradesToPublish []Trade) {
	numOfOrders := len(os)
	numOfTrades := len(tradesToPublish)
	tradesAndOrdersMsg := tradesAndOrders{height: height, timestamp: timestamp, numOfMsgs: numOfTrades + numOfOrders}
	if numOfOrders > 0 {
		tradesAndOrdersMsg.orders = orders{numOfOrders, os}
	}
	if numOfTrades > 0 {
		tradesAndOrdersMsg.trades = trades{numOfTrades, tradesToPublish}
	}

	if msg, err := marshal(&tradesAndOrdersMsg, tradesAndOrdersTpe); err == nil {
		kafkaMsg := publisher.prepareMessage(publisher.config.OrderUpdatesTopic, strconv.FormatInt(height, 10), timestamp, tradesAndOrdersTpe, msg)
		if partition, offset, err := publisher.producers[publisher.config.OrderUpdatesTopic].SendMessage(kafkaMsg); err == nil {
			Logger.Debug("published tradesAndOrders", "tradesAndOrders", tradesAndOrdersMsg.String(), "offset", offset, "partition", partition)
		} else {
			Logger.Error("failed to publish tradesAndOrders", "tradesAndOrders", tradesAndOrdersMsg.String(), "err", err)
		}
	} else {
		Logger.Error("failed to publish tradesAndOrders", "tradesAndOrders", tradesAndOrdersMsg.String(), "err", err)
	}
}

func (publisher *MarketDataPublisher) publishAccount(height int64, timestamp int64, accountsToPublish map[string]Account) {
	numOfMsgs := len(accountsToPublish)

	idx := 0
	accs := make([]Account, numOfMsgs, numOfMsgs)
	for _, acc := range accountsToPublish {
		accs[idx] = acc
		idx++
	}
	accountsMsg := accounts{height, numOfMsgs, accs}
	if msg, err := marshal(&accountsMsg, accountsTpe); err == nil {
		kafkaMsg := publisher.prepareMessage(
			publisher.config.AccountBalanceTopic,
			strconv.FormatInt(height, 10),
			timestamp,
			accountsTpe,
			msg)
		if partition, offset, err := publisher.producers[publisher.config.AccountBalanceTopic].SendMessage(kafkaMsg); err == nil {
			Logger.Debug("published accounts", "accounts", accountsMsg.String(), "offset", offset, "partition", partition)
		} else {
			Logger.Error("failed to publish accounts", "accounts", accountsMsg.String(), "err", err)
		}
	} else {
		Logger.Error("failed to publish accounts", "accounts", accountsMsg.String(), "err", err)
	}
}

// collect all changed books according to published order status
func (publisher *MarketDataPublisher) filterChangedOrderBooksByOrders(
	ordersToPublish []order,
	latestPriceLevels orderPkg.ChangedPriceLevelsMap) orderPkg.ChangedPriceLevelsMap {
	var res = make(orderPkg.ChangedPriceLevelsMap)
	for _, o := range ordersToPublish {
		if _, ok := latestPriceLevels[o.symbol]; !ok {
			continue
		}
		if _, ok := res[o.symbol]; !ok {
			res[o.symbol] = orderPkg.ChangedPriceLevelsPerSymbol{make(map[int64]int64), make(map[int64]int64)}
		}

		switch o.side {
		case orderPkg.Side.BUY:
			// TODO(#66): code clean up - here we rely on special implementation that for orders
			// that not generated from trade (like New, Cancel) the lastExecutedPrice is original price (rather than 0)
			if qty, ok := latestPriceLevels[o.symbol].Buys[o.lastExecutedPrice]; ok {
				res[o.symbol].Buys[o.lastExecutedPrice] = qty
			} else {
				res[o.symbol].Buys[o.lastExecutedPrice] = 0
			}
		case orderPkg.Side.SELL:
			if qty, ok := latestPriceLevels[o.symbol].Sells[o.lastExecutedPrice]; ok {
				res[o.symbol].Sells[o.lastExecutedPrice] = qty
			} else {
				res[o.symbol].Sells[o.lastExecutedPrice] = 0
			}
		}
	}
	return res
}

func (publisher *MarketDataPublisher) publishOrderBookData(height int64, timestamp int64, changedPriceLevels orderPkg.ChangedPriceLevelsMap) {
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

	books := books{height, timestamp, len(deltas), deltas}
	if msg, err := marshal(&books, booksTpe); err == nil {
		kafkaMsg := publisher.prepareMessage(publisher.config.OrderBookTopic, strconv.FormatInt(height, 10), timestamp, booksTpe, msg)
		if partition, offset, err := publisher.producers[publisher.config.OrderBookTopic].SendMessage(kafkaMsg); err == nil {
			Logger.Debug("published books", "books", books.String(), "offset", offset, "partition", partition)
		} else {
			Logger.Error("failed to publish books", "books", books.String(), "err", err)
		}
	} else {
		Logger.Error("failed to publish books", "books", books.String(), "err", err)
	}
}

func (publisher *MarketDataPublisher) newProducers() (config *sarama.Config, err error) {
	config = sarama.NewConfig()
	config.Version = sarama.MaxVersion
	if config.ClientID, err = os.Hostname(); err != nil {
		return
	}

	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 20
	config.Producer.Compression = sarama.CompressionNone

	// This MIGHT be kafka java client's equivalent max.in.flight.requests.per.connection
	// to make sure messages won't out-of-order
	// Refer: https://github.com/Shopify/sarama/issues/718
	config.Net.MaxOpenRequests = 1

	if publisher.config.PublishOrderUpdates {
		if _, ok := publisher.producers[publisher.config.OrderUpdatesTopic]; !ok {
			publisher.producers[publisher.config.OrderUpdatesTopic], err =
				sarama.NewSyncProducer([]string{publisher.config.OrderUpdatesKafka}, config)
		}
		if err != nil {
			Logger.Error("failed to create order updates producer", "err", err)
			return
		}
	}
	if publisher.config.PublishOrderBook {
		if _, ok := publisher.producers[publisher.config.OrderBookTopic]; !ok {
			publisher.producers[publisher.config.OrderBookTopic], err =
				sarama.NewSyncProducer([]string{publisher.config.OrderBookKafka}, config)
		}
		if err != nil {
			Logger.Error("failed to create order book producer", "err", err)
			return
		}
	}
	if publisher.config.PublishAccountBalance {
		if _, ok := publisher.producers[publisher.config.AccountBalanceTopic]; !ok {
			publisher.producers[publisher.config.AccountBalanceTopic], err =
				sarama.NewSyncProducer([]string{publisher.config.AccountBalanceKafka}, config)
		}
		if err != nil {
			Logger.Error("failed to create account balance producer", "err", err)
			return
		}
	}
	return
}

func (publisher *MarketDataPublisher) prepareMessage(
	topic string,
	msgId string,
	timeStamp int64,
	msgTpe msgType,
	message []byte) *sarama.ProducerMessage {
	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Partition: -1,
		Key:       sarama.StringEncoder(fmt.Sprintf("%s_%d_%s", msgId, timeStamp, msgTpe.String())),
		Value:     sarama.ByteEncoder(message),
	}

	return msg
}
