package log

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

func TestBasename(t *testing.T) {
	require := require.New(t)
	cases := map[string]string{
		"github.com/sirupsen/logrus/entry.go":   "logrus/entry.go",
		"gopkg.in/src-d/go-log.v1/logger.go":    "go-log.v1/logger.go",
		"gopkg.in/src-d/go-log.v0/logger.go":    "go-log.v0/logger.go",
		"gopkg.in/src-d/go-log.v0/main/main.go": "main/main.go",
		"file.go": "file.go",
		"":        "",
	}

	for k, v := range cases {
		require.Equalf(v, basename(k), k)
	}
}

func TestHasPrefix(t *testing.T) {
	require := require.New(t)
	cases := map[string]struct {
		prefix []string
		ok     bool
	}{
		"github.com/sirupsen/logrus/entry.go":   {[]string{"logrus"}, true},
		"gopkg.in/src-d/go-log.v1/logger.go":    {[]string{"go-log"}, true},
		"gopkg.in/src-d/go-log.v0/logger.go":    {[]string{"go-log"}, true},
		"gopkg.in/src-d/go-log.v0/main/main.go": {[]string{"go-log"}, false},
		"file.go": {[]string{"go-log", "logrus"}, false},
		"":        {[]string{"go-log", "logrus"}, false},
	}

	for k, v := range cases {
		file := basename(k)
		require.Equalf(v.ok, hasPrefix(file, v.prefix...), "path: %v, basename: %v", k, file)
	}
}

func TestCaller(t *testing.T) {
	require := require.New(t)

	hook := &filenameHook{
		field:      "source",
		skipframes: 1,
		skipnames:  []string{"logrus"},
		levels:     logrus.AllLevels,
		formatter: func(file string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		},
	}

	file, _ := hook.caller()
	require.Equal("go-log.v1/filename_test.go", file)
}

func TestCaller_IgnoreInternalCalls(t *testing.T) {
	require := require.New(t)
	hook := newFilenameHook(logrus.AllLevels...)

	file, _ := hook.caller()
	require.Equal(unknownFilename, file)
}
