package log

import (
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	require := require.New(t)

	os.Setenv("LOG_LEVEL", "DEBUG")

	l := New(nil)

	logger, ok := l.(*logger)
	require.True(ok)
	require.Equal(logrus.DebugLevel, logger.Entry.Logger.Level)
}

func TestWith(t *testing.T) {
	require := require.New(t)

	l := With(Fields{"foo": "bar"})

	logger, ok := l.(*logger)
	require.True(ok)
	require.Equal(logrus.Fields{"foo": "bar"}, logger.Entry.Data)
}

func TestInfof_Lazy(t *testing.T) {
	require := require.New(t)

	Infof("foo")
	require.NotNil(DefaultLogger)
}

func TestInfof(t *testing.T) {
	require := require.New(t)

	m := NewMockLogger()
	DefaultLogger = m

	Infof("foo")
	require.Equal(m.calledMethods["Infof"], "foo")
}

func TestDebugf(t *testing.T) {
	require := require.New(t)

	m := NewMockLogger()
	DefaultLogger = m

	Debugf("foo")
	require.Equal(m.calledMethods["Debugf"], "foo")
}

func TestWarningf(t *testing.T) {
	require := require.New(t)

	m := NewMockLogger()
	DefaultLogger = m

	Warningf("foo")
	require.Equal(m.calledMethods["Warningf"], "foo")
}
func TestErrorf(t *testing.T) {
	require := require.New(t)

	m := NewMockLogger()
	DefaultLogger = m

	Errorf(fmt.Errorf("foo"), "bar")
	require.Equal(m.calledMethods["Errorf"], "bar")
}

type MockLogger struct {
	calledMethods map[string]interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		calledMethods: make(map[string]interface{}, 0),
	}
}

func (l *MockLogger) New(f Fields) Logger {
	l.calledMethods["New"] = f
	return nil
}

func (l *MockLogger) With(f Fields) Logger {
	l.calledMethods["With"] = f
	return nil
}

func (l *MockLogger) Debugf(format string, args ...interface{}) {
	l.calledMethods["Debugf"] = format

}

func (l *MockLogger) Infof(format string, args ...interface{}) {
	l.calledMethods["Infof"] = format

}

func (l *MockLogger) Warningf(format string, args ...interface{}) {
	l.calledMethods["Warningf"] = format

}

func (l *MockLogger) Errorf(err error, format string, args ...interface{}) {
	l.calledMethods["Errorf"] = format

}
