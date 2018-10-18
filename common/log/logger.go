package log

import (
	"os"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

var (
	fileWriter *AsyncFileWriter
	logger     tmlog.Logger
)

func init() {
	logger = NewConsoleLogger()
}

func InitLogger(l tmlog.Logger) {
	// TODO: close log file when node stopped
	logger = l
}

func NewConsoleLogger() tmlog.Logger {
	return tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
}

func NewAsyncFileLogger(filePath string, buffSize int64) tmlog.Logger {
	if fileWriter != nil {
		fileWriter.Stop()
	}

	fileWriter = NewAsyncFileWriter(filePath, buffSize)
	fileWriter.Start()

	return tmlog.NewTMLogger(fileWriter)
}

func Debug(msg string, keyvals ...interface{}) {
	logger.Debug(msg, keyvals...)
}

func Info(msg string, keyvals ...interface{}) {
	logger.Info(msg, keyvals...)
}

func Error(msg string, keyvals ...interface{}) {
	logger.Error(msg, keyvals...)
}

func With(keyvals ...interface{}) tmlog.Logger {
	return logger.With(keyvals...)
}
