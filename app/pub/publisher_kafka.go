package pub

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/deathowl/go-metrics-prometheus"
	"github.com/eapache/go-resiliency/breaker"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
)

const (
	KafkaBrokerSep = ";"
)

type KafkaMarketDataPublisher struct {
	producers map[string]sarama.SyncProducer // topic -> producer
}

func (publisher *KafkaMarketDataPublisher) newProducers() (config *sarama.Config, err error) {
	config = sarama.NewConfig()
	config.Version = sarama.MaxVersion
	if config.ClientID, err = os.Hostname(); err != nil {
		return
	}

	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.MaxMessageBytes = 100 * 1024 * 1024 // TODO(#66): 100M, same with QA environment, make this configurable
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 20
	config.Producer.Compression = sarama.CompressionNone

	// This MIGHT be kafka java client's equivalent max.in.flight.requests.per.connection
	// to make sure messages won't out-of-order
	// Refer: https://github.com/Shopify/sarama/issues/718
	config.Net.MaxOpenRequests = 1

	if cfg.PublishOrderUpdates {
		if _, ok := publisher.producers[cfg.OrderUpdatesTopic]; !ok {
			publisher.producers[cfg.OrderUpdatesTopic], err =
				publisher.connectWithRetry(strings.Split(cfg.OrderUpdatesKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create order updates producer", "err", err)
			return
		}
	}
	if cfg.PublishOrderBook {
		if _, ok := publisher.producers[cfg.OrderBookTopic]; !ok {
			publisher.producers[cfg.OrderBookTopic], err =
				publisher.connectWithRetry(strings.Split(cfg.OrderBookKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create order book producer", "err", err)
			return
		}
	}
	if cfg.PublishAccountBalance {
		if _, ok := publisher.producers[cfg.AccountBalanceTopic]; !ok {
			publisher.producers[cfg.AccountBalanceTopic], err =
				publisher.connectWithRetry(strings.Split(cfg.AccountBalanceKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create account balance producer", "err", err)
			return
		}
	}
	if cfg.PublishBlockFee {
		if _, ok := publisher.producers[cfg.BlockFeeTopic]; !ok {
			publisher.producers[cfg.BlockFeeTopic], err =
				publisher.connectWithRetry(strings.Split(cfg.BlockFeeKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create blockfee producer", "err", err)
			return
		}
	}
	return
}

func (publisher *KafkaMarketDataPublisher) prepareMessage(
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

func (publisher *KafkaMarketDataPublisher) publish(avroMessage AvroMsg, tpe msgType, height, timestamp int64) {
	var topic string
	switch tpe {
	case booksTpe:
		topic = cfg.OrderBookTopic
	case accountsTpe:
		topic = cfg.AccountBalanceTopic
	case tradesAndOrdersTpe:
		topic = cfg.OrderUpdatesTopic
	case blockFeeTpe:
		topic = cfg.BlockFeeTopic
	}

	if msg, err := marshal(avroMessage, tpe); err == nil {
		kafkaMsg := publisher.prepareMessage(topic, strconv.FormatInt(height, 10), timestamp, tpe, msg)
		if partition, offset, err := publisher.publishWithRetry(kafkaMsg, topic); err == nil {
			Logger.Info("published", "topic", topic, "msg", avroMessage.String(), "offset", offset, "partition", partition)
		} else {
			Logger.Error("failed to publish", "topic", topic, "msg", avroMessage.String(), "err", err)
		}
	} else {
		Logger.Error("failed to publish", "topic", topic, "msg", avroMessage.String(), "err", err)
	}
}

func (publisher *KafkaMarketDataPublisher) Stop() {
	Logger.Debug("start to stop KafkaMarketDataPublisher")
	IsLive = false

	close(ToPublishCh)
	if ToRemoveOrderIdCh != nil {
		close(ToRemoveOrderIdCh)
	}

	for topic, producer := range publisher.producers {
		// nil check because this method would be called when we failed to create producer
		if producer != nil {
			if err := producer.Close(); err != nil {
				Logger.Error("failed to stop producer for topic", "topic", topic, "err", err)
			}
		}
	}
	Logger.Debug("finished stop KafkaMarketDataPublisher")
}

// endlessly retry on retriable errors, the abnormal situation should be reported by prometheus alarm
func (publisher *KafkaMarketDataPublisher) connectWithRetry(
	hostports []string,
	config *sarama.Config) (producer sarama.SyncProducer, err error) {
	backOffInSeconds := time.Duration(1)

	for {
		if producer, err = sarama.NewSyncProducer(hostports, config); err == sarama.ErrOutOfBrokers || err == breaker.ErrBreakerOpen {
			backOffInSeconds <<= 1
			Logger.Error("encountered retriable error, retrying...", "after", backOffInSeconds, "err", err)
			time.Sleep(backOffInSeconds * time.Second)
		} else {
			return
		}
	}
}

// endlessly retry on retriable errors, the abnormal situation should be reported by prometheus alarm
func (publisher *KafkaMarketDataPublisher) publishWithRetry(
	message *sarama.ProducerMessage,
	topic string) (partition int32, offset int64, err error) {
	backOffInSeconds := time.Duration(1)

	for {
		if partition, offset, err = publisher.producers[topic].SendMessage(message); err == sarama.ErrOutOfBrokers || err == breaker.ErrBreakerOpen {
			backOffInSeconds <<= 1
			Logger.Error("encountered retriable error, retrying...", "after", backOffInSeconds, "err", err)
			time.Sleep(backOffInSeconds * time.Second)
		} else {
			return
		}
	}
}

func NewKafkaMarketDataPublisher(
	logger log.Logger,
	config *config.PublicationConfig,
	metrics *Metrics) (publisher *KafkaMarketDataPublisher) {
	sarama.Logger = saramaLogger{}
	publisher = &KafkaMarketDataPublisher{
		producers: make(map[string]sarama.SyncProducer),
	}

	if err := setup(logger, config, metrics, publisher); err != nil {
		publisher.Stop()
		logger.Error("Cannot start up market data kafka publisher", "err", err)
		panic(err)
	}

	if saramaCfg, err := publisher.newProducers(); err != nil {
		Logger.Error("failed to create new kafka producer", "err", err)
		panic(err)
	} else {
		// we have to use the same prometheus registerer with tendermint
		// so that we can share same host:port for prometheus daemon
		prometheusRegistry := prometheus.DefaultRegisterer
		metricsRegistry := saramaCfg.MetricRegistry
		pClient := prometheusmetrics.NewPrometheusProvider(
			metricsRegistry,
			"",
			"publication",
			prometheusRegistry,
			1*time.Second)
		go pClient.UpdatePrometheusMetrics()
	}

	return publisher
}
