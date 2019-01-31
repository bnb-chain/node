package pub

import (
	"fmt"

	"github.com/binance-chain/node/common/log"
)

// this delegate sarama.StdLogger to common/log.logger
// the concrete implementation of this type is borrowed from same name methods of go std Logger
type saramaLogger struct{}

func (slogger saramaLogger) Print(v ...interface{}) {
	log.Debug(fmt.Sprint(v...), "module", "sarama")
}

func (slogger saramaLogger) Printf(format string, v ...interface{}) {
	log.Debug(fmt.Sprintf(format, v...), "module", "sarama")
}

func (slogger saramaLogger) Println(v ...interface{}) {
	log.Debug(fmt.Sprintln(v...), "module", "sarama")
}
