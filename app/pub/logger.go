package pub

import (
	"fmt"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

// this delegate sarama.StdLogger to common/log.logger
// the concrete implementation of this type is borrowed from same name methods of go std Logger
type saramaLogger struct {
	tmlog.Logger
}

func (slogger saramaLogger) Print(v ...interface{}) {
	slogger.Debug(fmt.Sprint(v...))
}

func (slogger saramaLogger) Printf(format string, v ...interface{}) {
	slogger.Debug(fmt.Sprintf(format, v...))
}

func (slogger saramaLogger) Println(v ...interface{}) {
	slogger.Debug(fmt.Sprintln(v...))
}
