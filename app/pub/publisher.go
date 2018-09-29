package pub

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	"github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/common/utils"
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
	Logger log.Logger
)

type MarketDataPublisher struct {
	ToPublishCh       chan BlockInfoToPublish
	ToRemoveOrderIdCh chan string   // order ids to remove from keeper.OrderChangesMap
	RemoveDoneCh      chan struct{} // order ids to remove for this block is done
	IsLive            bool          // TODO(#66): thread safty: is EndBlocker and Init are call in same thread?

	config    *config.PublicationConfig
	producers map[string]sarama.SyncProducer // topic -> producer
}

func (publisher *MarketDataPublisher) Init(config *config.PublicationConfig, logger log.Logger) (err error) {
	sarama.Logger = saramaLogger{}
	Logger = logger
	publisher.config = config
	publisher.producers = make(map[string]sarama.SyncProducer)

	if config, err := publisher.newProducers(); err != nil {
		Logger.Error("failed to create new kafka producer", "err", err)
		return err
	} else {
		// we have to use the same prometheus registerer with tendermint so that we can share same host:port for prometheus daemon
		prometheusRegistry := prometheus.DefaultRegisterer
		metricsRegistry := config.MetricRegistry
		pClient := prometheusmetrics.NewPrometheusProvider(metricsRegistry, "", "publication", prometheusRegistry, 1*time.Second)
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
	Logger.Info("start to stop MarketDataPublisher")
	publisher.IsLive = false

	close(publisher.ToPublishCh)
	close(publisher.ToRemoveOrderIdCh)
	close(publisher.RemoveDoneCh)

	for topic, producer := range publisher.producers {
		// nil check because this method would be called when we failed to create producer
		if producer != nil {
			if err := producer.Close(); err != nil {
				Logger.Error("faid to stop producer for topic", "topic", topic, "err", err)
			}
		}
	}
	Logger.Info("finished stop MarketDataPublisher")
}

func (publisher *MarketDataPublisher) ShouldPublish() bool {
	return publisher.IsLive && publisher.config.ShouldPublishAny()
}

func (publisher *MarketDataPublisher) publish() {
	for marketData := range publisher.ToPublishCh {
		// Implementation note: publication order are important here,
		// DEX query service team relies on the fact that we publish orders before trades so that
		// they can assign buyer/seller address into trade before persist into DB
		var ordersToPublish []order
		if publisher.config.PublishOrderUpdates || publisher.config.PublishOrderBook {
			ordersToPublish = publisher.collectFilledOrdersFromTrade(
				&marketData.tradesToPublish,
				marketData.orderChanges,
				marketData.orderChangesMap,
				marketData.timestamp)
		}
		publisher.RemoveDoneCh <- struct{}{}

		if publisher.config.PublishOrderUpdates {
			Logger.Info("start to publish all orders")
			publisher.publishOrderUpdates(
				marketData.height,
				marketData.timestamp,
				ordersToPublish,
				marketData.tradesToPublish)
		}

		if publisher.config.PublishAccountBalance {
			Logger.Info("start to publish all changed accounts")
			publisher.publishAccount(marketData.height, marketData.timestamp, marketData.accounts)
		}

		if publisher.config.PublishOrderBook {
			Logger.Info("start to publish changed order books")
			changedPrices := publisher.collectChangedOrderBooksFromOrders(
				&ordersToPublish,
				marketData.latestPricesLevels)
			publisher.publishOrderBookData(marketData.height, marketData.timestamp, changedPrices)
		}
	}
}

// we collect OrderPart here to make matcheng module independent
func (publisher *MarketDataPublisher) collectFilledOrdersFromTrade(
	trades *[]Trade,
	orderChanges orderPkg.OrderChanges,
	orderChangesMap orderPkg.OrderChangesMap,
	timestamp int64) (ordersToPublish []order) {
	ordersToPublish = make([]order, 0)
	canceledToPublish := make([]order, 0)

	// collect orders (new, cancel, ioc-no-fill, expire) from orderChanges
	for idx, o := range orderChanges {
		orderToPublish := order{
			o.OrderMsg.Symbol,
			o.Tpe,
			o.OrderMsg.Id,
			"",
			o.OrderMsg.Sender.String(),
			o.OrderMsg.Side,
			orderPkg.OrderType.LIMIT,
			o.OrderMsg.Price,
			o.OrderMsg.Quantity,
			0,
			0,
			o.CumQty,
			o.CumQuoteAssetQty,
			orderChangesMap[o.OrderMsg.Id].Fee,
			orderChangesMap[o.OrderMsg.Id].FeeAsset,
			o.CreationTime(),
			timestamp,
			o.OrderMsg.TimeInForce,
			orderPkg.NEW,
			o.TxHash,
		}
		if o.Tpe == orderPkg.Ack {
			o.SetCreationTime(timestamp)
			orderChanges[idx].SetCreationTime(timestamp)
			orderChangesMap[o.OrderMsg.Id].SetCreationTime(timestamp)
			ordersToPublish = append(ordersToPublish, orderToPublish)
		} else {
			canceledToPublish = append(canceledToPublish, orderToPublish)
			Logger.Debug(
				"going to delete order from order changes map",
				"orderId", o.OrderMsg.Id, "reason", orderToPublish.status)
			publisher.ToRemoveOrderIdCh <- o.OrderMsg.Id
		}
	}

	// collect orders from trades
	for _, t := range *trades {
		if o, exists := orderChangesMap[t.Bid]; exists {
			ordersToPublish = append(ordersToPublish, publisher.collectFilledOrderFromTrade(t, o, timestamp))
		} else {
			Logger.Error("failed to resolve order information from orderChangesMap", "orderId", t.Bid)
		}

		if o, exists := orderChangesMap[t.Sid]; exists {
			ordersToPublish = append(ordersToPublish, publisher.collectFilledOrderFromTrade(t, o, timestamp))
		} else {
			Logger.Error("failed to resolve order information from orderChangesMap", "orderId", t.Sid)
		}
	}

	return append(ordersToPublish, canceledToPublish...)
}

func (publisher *MarketDataPublisher) collectFilledOrderFromTrade(
	t Trade,
	o *orderPkg.OrderChange,
	timestamp int64) order {

	// accumulate numbers because we need know the leaves and cum quantities
	// for expired order
	o.CumQty += t.Qty
	o.CumQuoteAssetQty += utils.CalBigNotional(t.Qty, t.Price) //TODO(#66): confirm with danjun this value is right

	var status orderPkg.ChangeType
	if o.CumQty == o.OrderMsg.Quantity {
		status = orderPkg.FullyFill
	} else {
		status = orderPkg.PartialFill
	}
	var fee int64
	var feeAsset string
	if o.OrderMsg.Side == orderPkg.Side.BUY {
		fee = t.Bfee
		feeAsset = t.BfeeAsset
	} else {
		fee = t.Sfee
		feeAsset = t.SfeeAsset
	}
	res := order{
		o.OrderMsg.Symbol,
		status,
		o.OrderMsg.Id,
		t.Id,
		o.OrderMsg.Sender.String(),
		o.OrderMsg.Side,
		orderPkg.OrderType.LIMIT,
		o.OrderMsg.Price,
		o.OrderMsg.Quantity,
		t.Price,
		t.Qty,
		o.CumQty,
		o.CumQuoteAssetQty,
		fee,
		feeAsset,
		o.CreationTime(),
		timestamp,
		o.OrderMsg.TimeInForce,
		orderPkg.NEW,
		o.TxHash,
	}
	if status == orderPkg.FullyFill {
		Logger.Debug("going to delete order from order changes map because of fully fill", "orderId", o.OrderMsg.Id)
		publisher.ToRemoveOrderIdCh <- o.OrderMsg.Id
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
			Logger.Info("published tradesAndOrders", "tradesAndOrders", tradesAndOrdersMsg.String(), "offset", offset, "partition", partition)
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
		kafkaMsg := publisher.prepareMessage(publisher.config.AccountBalanceTopic, strconv.FormatInt(height, 10), timestamp, accountsTpe, msg)
		if partition, offset, err := publisher.producers[publisher.config.AccountBalanceTopic].SendMessage(kafkaMsg); err == nil {
			Logger.Info("published accounts", "accounts", accountsMsg.String(), "offset", offset, "partition", partition)
		} else {
			Logger.Error("failed to publish accounts", "accounts", accountsMsg.String(), "err", err)
		}
	} else {
		Logger.Error("failed to publish accounts", "accounts", accountsMsg.String(), "err", err)
	}
}

// collect all changed books according to published order status
func (publisher *MarketDataPublisher) collectChangedOrderBooksFromOrders(ordersToPublish *[]order, latestPriceLevels orderPkg.ChangedPriceLevels) orderPkg.ChangedPriceLevels {
	var res = make(orderPkg.ChangedPriceLevels)
	for _, o := range *ordersToPublish {
		if _, ok := latestPriceLevels[o.symbol]; !ok {
			continue
		}
		if _, ok := res[o.symbol]; !ok {
			res[o.symbol] = orderPkg.ChangedPriceLevelsPerSymbol{make(map[int64]int64), make(map[int64]int64)}
		}

		switch o.side {
		case orderPkg.Side.BUY:
			// TODO(#66): code clean up - here we rely on special implementation that for orders that not generated from trade (like New, Cancel) the lastExecutedPrice is original price (rather than 0)
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

func (publisher *MarketDataPublisher) publishOrderBookData(height int64, timestamp int64, changedPriceLevels orderPkg.ChangedPriceLevels) {
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
			Logger.Info("published books", "books", books.String(), "offset", offset, "partition", partition)
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
