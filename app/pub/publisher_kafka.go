package pub

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/eapache/go-resiliency/breaker"
	"github.com/linkedin/goavro"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	KafkaBrokerSep  = ";"
	essentialLogDir = "essential"
)

type KafkaMarketDataPublisher struct {
	booksCodec            *goavro.Codec
	accountCodec          *goavro.Codec
	executionResultsCodec *goavro.Codec
	blockFeeCodec         *goavro.Codec
	transfersCodec        *goavro.Codec
	blockCodec            *goavro.Codec
	stakingCodec          *goavro.Codec
	distributionCodec     *goavro.Codec
	slashingCodec         *goavro.Codec
	crossTransferCodec    *goavro.Codec
	mirrorCodec           *goavro.Codec
	sideProposalCodec     *goavro.Codec
	breatheBlockCodec     *goavro.Codec

	failFast         bool
	essentialLogPath string                         // the path (default to db dir) we write essential file to make up data on kafka error
	producers        map[string]sarama.SyncProducer // topic -> producer
}

func NewTLSConfig(clientCertFile, clientKeyFile, caCertFile string) (*tls.Config, error) {
	tlsConfig := tls.Config{}
	if len(clientCertFile) != 0 || len(clientKeyFile) != 0 {
		cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return &tlsConfig, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if len(caCertFile) != 0 {
		caCert, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			return &tlsConfig, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	tlsConfig.BuildNameToCertificate()
	return &tlsConfig, nil
}

func (publisher *KafkaMarketDataPublisher) newProducers() (config *sarama.Config, err error) {
	version, err := sarama.ParseKafkaVersion(Cfg.KafkaVersion)
	if err != nil {
		return nil, fmt.Errorf("%v. Please choose a version within: %s", err, sarama.SupportedVersions)
	}
	supported := false
	for _, supportedVer := range sarama.SupportedVersions {
		if version == supportedVer {
			supported = true
			break
		}
	}
	if !supported {
		return nil, fmt.Errorf("kafka version in app.toml is not supported. Please choose a version within: %s", sarama.SupportedVersions)
	}

	config = sarama.NewConfig()
	config.Version = version
	if config.ClientID, err = os.Hostname(); err != nil {
		return
	}

	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.MaxMessageBytes = 100 * 1024 * 1024 // TODO(#66): 100M, same with QA environment, make this configurable
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 20
	config.Producer.Compression = sarama.CompressionGZIP

	if Cfg.Auth {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		config.Net.SASL.User = Cfg.KafkaUserName
		config.Net.SASL.Password = Cfg.KafkaPassword
	}

	if Cfg.TLS {
		config.Net.TLS.Enable = true
		if len(Cfg.ClientKeyPath) != 0 || len(Cfg.ClientCertPath) != 0 || len(Cfg.CaCertPath) == 0 {
			tlsConfig, err := NewTLSConfig(Cfg.ClientCertPath,
				Cfg.ClientKeyPath, Cfg.CaCertPath)
			if err != nil {
				return nil, err
			}
			config.Net.TLS.Config = tlsConfig
		}
		Logger.Info("Enabled TLS")
	}

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
	if Cfg.PublishTransfer {
		if _, ok := publisher.producers[Cfg.TransferTopic]; !ok {
			publisher.producers[Cfg.TransferTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.TransferKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create transfers producer", "err", err)
			return
		}
	}
	if Cfg.PublishBlock {
		if _, ok := publisher.producers[Cfg.BlockTopic]; !ok {
			publisher.producers[Cfg.BlockTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.BlockKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create blocks producer", "err", err)
			return
		}
	}
	if Cfg.PublishStaking {
		if _, ok := publisher.producers[Cfg.StakingTopic]; !ok {
			publisher.producers[Cfg.StakingTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.StakingKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create staking producer", "err", err)
			return
		}
	}
	if Cfg.PublishDistributeReward {
		if _, ok := publisher.producers[Cfg.DistributeRewardTopic]; !ok {
			publisher.producers[Cfg.DistributeRewardTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.DistributeRewardKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create distribution producer", "err", err)
			return
		}
	}
	if Cfg.PublishSlashing {
		if _, ok := publisher.producers[Cfg.SlashingTopic]; !ok {
			publisher.producers[Cfg.SlashingTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.SlashingKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create slashing producer", "err", err)
			return
		}
	}
	if Cfg.PublishCrossTransfer {
		if _, ok := publisher.producers[Cfg.CrossTransferTopic]; !ok {
			publisher.producers[Cfg.CrossTransferTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.CrossTransferKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create crossTransfer producer", "err", err)
			return
		}
	}
	if Cfg.PublishMirror {
		if _, ok := publisher.producers[Cfg.MirrorTopic]; !ok {
			publisher.producers[Cfg.MirrorTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.MirrorKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create mirror producer", "err", err)
			return
		}
	}
	if Cfg.PublishSideProposal {
		if _, ok := publisher.producers[Cfg.SideProposalTopic]; !ok {
			publisher.producers[Cfg.SideProposalTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.SideProposalKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create crossTransfer producer", "err", err)
			return
		}
	}
	if Cfg.PublishBreatheBlock {
		if _, ok := publisher.producers[Cfg.BreatheBlockTopic]; !ok {
			publisher.producers[Cfg.BreatheBlockTopic], err =
				publisher.connectWithRetry(strings.Split(Cfg.BreatheBlockKafka, KafkaBrokerSep), config)
		}
		if err != nil {
			Logger.Error("failed to create breathe block producer", "err", err)
			return
		}
	}
	return
}

// Key of kafka message contains following fields separated by underscore:
// 1. message_id
// 2. timestamp of message
// 3. type of value (multiple types of messages can be published for one kafka topic)
// 4. value's encoding schema version.
func (publisher *KafkaMarketDataPublisher) prepareMessage(
	topic string,
	msgId string,
	timeStamp int64,
	msgTpe msgType,
	message []byte) *sarama.ProducerMessage {
	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Partition: -1,
		Key:       sarama.StringEncoder(fmt.Sprintf("%s_%d_%s_%d", msgId, timeStamp, msgTpe.String(), latestSchemaVersions[msgTpe])),
		Value:     sarama.ByteEncoder(message),
	}

	return msg
}

func (publisher *KafkaMarketDataPublisher) publish(avroMessage AvroOrJsonMsg, tpe msgType, height, timestamp int64) {
	topic := publisher.resolveTopic(tpe)

	if msg, err := publisher.marshal(avroMessage, tpe); err == nil {
		kafkaMsg := publisher.prepareMessage(topic, strconv.FormatInt(height, 10), timestamp, tpe, msg)
		if partition, offset, err := publisher.publishWithRetry(kafkaMsg, topic); err == nil {
			Logger.Info("published", "topic", topic, "msg", avroMessage.String(), "offset", offset, "partition", partition)
		} else {
			Logger.Error("failed to publish, tring to log essential message", "topic", topic, "msg", avroMessage.String(), "err", err)
			if essMsg, ok := avroMessage.(EssMsg); ok {
				publisher.publishEssentialMsg(essMsg, topic, tpe, height, timestamp)
			}
			if publisher.failFast {
				panic(fmt.Sprintf("publish kafka message failed %v", err))
			}
		}
	} else {
		Logger.Error("failed to publish", "topic", topic, "msg", avroMessage.String(), "err", err)
	}
}

func (publisher KafkaMarketDataPublisher) publishEssentialMsg(essMsg EssMsg, topic string, tpe msgType, height, timestamp int64) {
	// First, publish an empty copy to make sure downstream service not hanging
	if msg, err := publisher.marshal(essMsg.EmptyCopy(), tpe); err == nil {
		kafkaMsg := publisher.prepareMessage(topic, strconv.FormatInt(height, 10), timestamp, tpe, msg)
		if partition, offset, err := publisher.publishWithRetry(kafkaMsg, topic); err == nil {
			// deliberately be Error level to trigger logging service elastic search alert
			Logger.Error("published empty msg", "topic", topic, "msg", essMsg.String(), "offset", offset, "partition", partition)
		} else {
			Logger.Error("failed to publish empty msg", "topic", topic, "msg", essMsg.String(), "err", err)
		}
	} else {
		Logger.Error("failed to publish empty msg", "topic", topic, "msg", essMsg.String(), "err", err)
	}

	// Second, log essential content of message to hard disk
	filePath := fmt.Sprintf("%s/%d_%s.log", publisher.essentialLogPath, height, tpe.String())
	toWrite := []byte(essMsg.EssentialMsg())
	if len(toWrite) != 0 {
		if err := ioutil.WriteFile(filePath, toWrite, 0644); err != nil {
			Logger.Error("failed to write essential log", "err", err)
		}
	}
}

func (publisher KafkaMarketDataPublisher) resolveTopic(tpe msgType) (topic string) {
	switch tpe {
	case booksTpe:
		topic = Cfg.OrderBookTopic
	case accountsTpe:
		topic = Cfg.AccountBalanceTopic
	case executionResultTpe:
		topic = Cfg.OrderUpdatesTopic
	case blockFeeTpe:
		topic = Cfg.BlockFeeTopic
	case transferTpe:
		topic = Cfg.TransferTopic
	case blockTpe:
		topic = Cfg.BlockTopic
	case stakingTpe:
		topic = Cfg.StakingTopic
	case distributionTpe:
		topic = Cfg.DistributeRewardTopic
	case slashingTpe:
		topic = Cfg.SlashingTopic
	case crossTransferTpe:
		topic = Cfg.CrossTransferTopic
	case mirrorTpe:
		topic = Cfg.MirrorTopic
	case sideProposalType:
		topic = Cfg.SideProposalTopic
	case breatheBlockTpe:
		topic = Cfg.BreatheBlockTopic
	}
	return
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
		if partition, offset, err = publisher.producers[topic].SendMessage(message); err != nil && publisher.isRetryableErr(err) {
			backOffInSeconds <<= 1
			backOffTime := backOffInSeconds * time.Second
			Logger.Error("encountered retryable error, retrying...", "after", backOffTime, "err", err)
			time.Sleep(backOffTime)
		} else {
			return
		}
	}
}

// these errors happens indicate there is something wrong with kafka cluster,
// we should not continue before the connectivity issue is gone
// otherwise, downstream services might missing blocks
func (publisher *KafkaMarketDataPublisher) isRetryableErr(err error) bool {
	return err == sarama.ErrOutOfBrokers ||
		err == breaker.ErrBreakerOpen ||
		strings.Contains(err.Error(), "i/o timeout") ||
		strings.Contains(err.Error(), "connection refused")
}

func (publisher *KafkaMarketDataPublisher) marshal(msg AvroOrJsonMsg, tpe msgType) ([]byte, error) {
	native := msg.ToNativeMap()
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
	case transferTpe:
		codec = publisher.transfersCodec
	case blockTpe:
		codec = publisher.blockCodec
	case stakingTpe:
		codec = publisher.stakingCodec
	case distributionTpe:
		codec = publisher.distributionCodec
	case slashingTpe:
		codec = publisher.slashingCodec
	case crossTransferTpe:
		codec = publisher.crossTransferCodec
	case mirrorTpe:
		codec = publisher.mirrorCodec
	case sideProposalType:
		codec = publisher.sideProposalCodec
	case breatheBlockTpe:
		codec = publisher.breatheBlockCodec
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
	} else if publisher.transfersCodec, err = goavro.NewCodec(transfersSchema); err != nil {
		return err
	} else if publisher.blockCodec, err = goavro.NewCodec(blockDatasSchema); err != nil {
		return err
	} else if publisher.stakingCodec, err = goavro.NewCodec(stakingSchema); err != nil {
		return err
	} else if publisher.distributionCodec, err = goavro.NewCodec(distributionSchema); err != nil {
		return err
	} else if publisher.slashingCodec, err = goavro.NewCodec(slashingSchema); err != nil {
		return err
	} else if publisher.crossTransferCodec, err = goavro.NewCodec(crossTransferSchema); err != nil {
		return err
	} else if publisher.mirrorCodec, err = goavro.NewCodec(mirrorSchema); err != nil {
		return err
	} else if publisher.sideProposalCodec, err = goavro.NewCodec(sideProposalsSchema); err != nil {
		return err
	} else if publisher.breatheBlockCodec, err = goavro.NewCodec(breatheBlockSchema); err != nil {
		return err
	}
	return nil
}

func NewKafkaMarketDataPublisher(
	logger log.Logger, dbDir string, failFast bool) (publisher *KafkaMarketDataPublisher) {

	sarama.Logger = saramaLogger{logger.With("module", "sarama")}
	publisher = &KafkaMarketDataPublisher{
		producers:        make(map[string]sarama.SyncProducer),
		essentialLogPath: filepath.Join(dbDir, essentialLogDir),
		failFast:         failFast,
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

	if err := common.EnsureDir(publisher.essentialLogPath, 0755); err != nil {
		logger.Error("failed to create essential log path", "err", err)
	}

	logger.Info("created kafka publisher", "elpath", publisher.essentialLogPath)
	return publisher
}
