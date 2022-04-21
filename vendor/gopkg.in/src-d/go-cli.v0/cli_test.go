package cli

import (
	"runtime"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/require"
)

func TestHelp(t *testing.T) {
	require := require.New(t)

	app := New("test", "0.1.0", "abc", "my test bin")
	require.NotNil(app)

	var (
		stdout, stderr string
		err            error
	)
	stdout = capturer.CaptureStdout(func() {
		stderr = capturer.CaptureStderr(func() {
			err = app.Run([]string{"test", "--help"})
		})
	})

	require.NoError(err)
	require.Empty(stderr)
	if runtime.GOOS == "windows" {
		require.Equal(`Usage:
  test [OPTIONS] <version>

my test bin

Help Options:
  /?          Show this help message
  /h, /help   Show this help message

Available commands:
  version  print version

`, stdout)
	} else {
		require.Equal(`Usage:
  test [OPTIONS] <version>

my test bin

Help Options:
  -h, --help  Show this help message

Available commands:
  version  print version

`, stdout)
	}
}

func TestHelpError(t *testing.T) {
	require := require.New(t)

	app := New("test", "0.1.0", "abc", "my test bin")
	require.NotNil(app)

	var (
		stdout, stderr string
		err            error
	)
	stdout = capturer.CaptureStdout(func() {
		stderr = capturer.CaptureStderr(func() {
			err = app.Run([]string{"test", "--bad-option"})
		})
	})

	require.Error(err)
	require.Empty(stdout)
	if runtime.GOOS == "windows" {
		require.Equal(`unknown flag `+"`"+`bad-option'
Usage:
  test [OPTIONS] <version>

my test bin

Help Options:
  /?          Show this help message
  /h, /help   Show this help message

Available commands:
  version  print version
`, stderr)
	} else {
		require.Equal(`unknown flag `+"`"+`bad-option'
Usage:
  test [OPTIONS] <version>

my test bin

Help Options:
  -h, --help  Show this help message

Available commands:
  version  print version
`, stderr)
	}
}

func TestAddCommandError(t *testing.T) {
	require := require.New(t)

	app := New("test", "0.1.0", "abc", "my test bin")
	require.NotNil(app)

	require.Panics(func() {
		app.AddCommand(nil)
	})

	require.Panics(func() {
		app.AddCommand(badCommander(42))
	})

	require.Panics(func() {
		app.AddCommand(struct {
			Command
			badCommander
		}{})
	})
}

type badCommander int

func (badCommander) Execute(args []string) error {
	return nil
}

func TestDefer(t *testing.T) {
	require := require.New(t)

	app := New("test", "0.1.0", "abc", "my test bin")
	require.NotNil(app)

	expected := []int{7, 6, 5, 4, 3, 2, 1, 0}
	var result []int

	for i := range expected {
		num := i
		app.Defer(func() {
			result = append(result, num)
		})
	}

	require.Nil(result)

	err := app.Run([]string{"test", "version"})
	require.NoError(err)
	require.Equal(expected, result)
}
