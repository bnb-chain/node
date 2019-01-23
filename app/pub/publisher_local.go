package pub

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/natefinch/lumberjack"

	"github.com/BiJie/BinanceChain/app/config"

	tmLogger "github.com/tendermint/tendermint/libs/log"
)

// Publish market data to local marketdata dir in bnbchaind home
// each message will be in json format one line in file
// file can be compressed and auto-rotated
type LocalMarketDataPublisher struct {
	producer *log.Logger
	tmLogger tmLogger.Logger
}

func (publisher *LocalMarketDataPublisher) publish(msg AvroOrJsonMsg, tpe msgType, height int64, timestamp int64) {
	if jsonBytes, err := json.Marshal(msg); err == nil {
		if err := publisher.producer.Output(2, fmt.Sprintln(string(jsonBytes))); err != nil {
			publisher.tmLogger.Error("failed to publish msg", "err", err, "height", height, "msg", msg.String())
		}
	} else {
		publisher.tmLogger.Error("failed to publish msg", "err", err, "height", height, "msg", msg.String())
	}
}

func (publisher *LocalMarketDataPublisher) Stop() {
	publisher.tmLogger.Info("local publisher stopped")
}

func NewLocalMarketDataPublisher(
	dataPath string,
	tmLogger tmLogger.Logger,
	config *config.PublicationConfig) (publisher *LocalMarketDataPublisher) {
	fileWriter := &lumberjack.Logger{
		Filename: fmt.Sprintf("%s/marketdata/marketdata.json", dataPath),
		MaxSize:  config.LocalMaxSize,
		MaxAge:   config.LocalMaxAge,
		Compress: true,
	}
	logger := log.New(fileWriter, "", 0)
	logger.SetOutput(fileWriter)
	publisher = &LocalMarketDataPublisher{
		logger,
		tmLogger,
	}

	return
}
