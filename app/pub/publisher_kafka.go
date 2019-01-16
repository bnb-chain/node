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
	"github.com/linkedin/goavro"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tendermint/tendermint/libs/log"
)

const (
	KafkaBrokerSep = ";"
)

type KafkaMarketDataPublisher struct {
	booksCodec            *goavro.Codec
	accountCodec          *goavro.Codec
	executionResultsCodec *goavro.Codec
	blockFeeCodec         *goavro.Codec

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
	config.Producer.Compression = sarama.CompressionGZIP

	// This MIGHT be kafka java client's equivalent max.in.flight.requests.per.connection
	// to make sure messages won't out-of-order
	// Refer: https://github.com/Shopify/sarama/issues/718
	config.Net.MaxOpenRequests = 1

	if Cfg.PublishOrderUpdates {
		if _, ok := publisher.producers[Cfg.OrderUpdatesTopic]; !ok {
			publisher.producers[Cfg.OrderUpdatesTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.OrderUpdatesKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create order updates producer", "err", err)
			return
		}
	}
	if Cfg.PublishOrderBook {
		if _, ok := publisher.producers[Cfg.OrderBookTopic]; !ok {
			publisher.producers[Cfg.OrderBookTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.OrderBookKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create order book producer", "err", err)
			return
		}
	}
	if Cfg.PublishAccountBalance {
		if _, ok := publisher.producers[Cfg.AccountBalanceTopic]; !ok {
			publisher.producers[Cfg.AccountBalanceTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.AccountBalanceKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create account balance producer", "err", err)
			return
		}
	}
	if Cfg.PublishBlockFee {
		if _, ok := publisher.producers[Cfg.BlockFeeTopic]; !ok {
			publisher.producers[Cfg.BlockFeeTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.BlockFeeKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create blockfee producer", "err", err)
			return
		}
	}
	if cfg.PublishTransfer {
		if _, ok := publisher.producers[cfg.TransferTopic]; !ok {
			publisher.producers[cfg.TransferTopic], err =
				publisher.connectWithRetry(strings.Split(cfg.TransferKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create transfers producer", "err", err)
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

func (publisher *KafkaMarketDataPublisher) publish(avroMessage AvroOrJsonMsg, tpe msgType, height, timestamp int64) {
	var topic string
	switch tpe {
	case booksTpe:
		topic = Cfg.OrderBookTopic
	case accountsTpe:
		topic = Cfg.AccountBalanceTopic
	case executionResultTpe:
		topic = Cfg.OrderUpdatesTopic
	case blockFeeTpe:
		topic = Cfg.BlockFeeTopic
	case transferType:
		topic = Cfg.TransferTopic
	}

	if msg, err := publisher.marshal(avroMessage, tpe); err == nil {
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

func (publisher *KafkaMarketDataPublisher) marshal(msg AvroOrJsonMsg, tpe msgType) ([]byte, error) {
	native := msg.ToNativeMap()
	Logger.Debug("msgDetail", "msg", native)
	var codec *goavro.Codec
	switch tpe {
	case accountsTpe:
		codec = publisher.accountCodec
	case booksTpe:
		codec = publisher.booksCodec
	case executionResultTpe:
		codec = publisher.executionResultsCodec
	case blockFeeTpe:
		codec = publisher.blockFeeCodec
	default:
		return nil, fmt.Errorf("doesn't support marshal kafka msg tpe: %s", tpe.String())
	}
	bb, err := codec.BinaryFromNative(nil, native)
	if err != nil {
		Logger.Error("failed to serialize message", "msg", msg, "err", err)
	}
	return bb, err
}

func (publisher *KafkaMarketDataPublisher) initAvroCodecs() (err error) {
	if publisher.executionResultsCodec, err = goavro.NewCodec(executionResultSchema); err != nil {
		return err
	} else if publisher.booksCodec, err = goavro.NewCodec(booksSchema); err != nil {
		return err
	} else if publisher.accountCodec, err = goavro.NewCodec(accountSchema); err != nil {
		return err
	} else if publisher.blockFeeCodec, err = goavro.NewCodec(blockfeeSchema); err != nil {
		return err
	}
	return nil
}

func NewKafkaMarketDataPublisher(
	logger log.Logger) (publisher *KafkaMarketDataPublisher) {

	sarama.Logger = saramaLogger{}
	publisher = &KafkaMarketDataPublisher{
		producers: make(map[string]sarama.SyncProducer),
	}

	if err := publisher.initAvroCodecs(); err != nil {
		Logger.Error("failed to initialize avro codec", "err", err)
		panic(err)
	}

	if saramaCfg, err := publisher.newProducers(); err != nil {
		logger.Error("failed to create new kafka producer", "err", err)
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
