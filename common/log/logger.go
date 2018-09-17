package log

import (
	"os"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

var (
	defaultFileWriter *AsyncFileWriter
	defaultLogger     tmlog.Logger
)

func init() {
	defaultLogger = tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
}

func Init(logger tmlog.Logger) {
	// TODO: close log file when node stopped
	defaultLogger = logger
}

func NewAsyncFileLogger(filePath string, buffSize int64) tmlog.Logger {
	if defaultFileWriter != nil {
		defaultFileWriter.Stop()
	}

	defaultFileWriter = NewAsyncFileWriter(filePath, buffSize)
	defaultFileWriter.Start()

	logger := tmlog.NewTMLogger(defaultFileWriter)
	return logger
}

func Debug(msg string, keyvals ...interface{}) {
	defaultLogger.Debug(msg, keyvals...)
}

func Info(msg string, keyvals ...interface{}) {
	defaultLogger.Info(msg, keyvals...)
}

func Error(msg string, keyvals ...interface{}) {
	defaultLogger.Error(msg, keyvals...)
}

func With(keyvals ...interface{}) tmlog.Logger {
	return defaultLogger.With(keyvals...)
}
