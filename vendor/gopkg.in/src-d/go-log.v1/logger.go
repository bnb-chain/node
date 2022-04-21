package log

import "github.com/sirupsen/logrus"

// Fields type, used to pass to `Logger.New`.
type Fields map[string]interface{}

// Logger represents a generic logger, based on logrus.Logger
type Logger interface {
	// New returns a copy of the current logger, adding the given Fields.
	New(Fields) Logger
	// With returns a copy of the current logger, adding the given Fields.
	// Alias of New, must be used when is chained with any message function.
	With(Fields) Logger
	// Debugf logs a message at level Debug.
	Debugf(format string, args ...interface{})
	// Infof logs a message at level Info.
	Infof(format string, args ...interface{})
	// Warningf logs a message at level Warning.
	Warningf(format string, args ...interface{})
	// Errorf logs an error with a message at level Error.
	Errorf(err error, format string, args ...interface{})
}

type logger struct {
	logrus.Entry
}

func (l *logger) New(f Fields) Logger {
	e := l.WithFields(logrus.Fields(f))
	return &logger{*e}
}

func (l *logger) With(f Fields) Logger { return l.New(f) }

func (l *logger) Errorf(err error, format string, args ...interface{}) {
	l.WithError(err).Errorf(format, args...)
}
