package cli

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type LogCommand struct {
	Command `name:"nop" short-description:"nop" long-description:"nop"`
}

func (c *LogCommand) Execute(args []string) error {
	return nil
}

func setupLogCommand(t *testing.T) *App {
	app := New("test", "", "", "")
	app.AddCommand(&LogCommand{})
	return app
}

func TestLogLevel(t *testing.T) {
	fixtures := []struct {
		level string
		err   bool
	}{{
		level: "info",
		err:   false,
	}, {
		level: "debug",
		err:   false,
	}, {
		level: "warning",
		err:   false,
	}, {
		level: "error",
		err:   false,
	}, {
		level: "other",
		err:   true,
	}}

	app := setupLogCommand(t)

	for _, fixture := range fixtures {
		t.Run(fixture.level, func(t *testing.T) {
			require := require.New(t)

			err := app.Run([]string{"test", "nop",
				fmt.Sprintf("--log-level=%s", fixture.level)})

			if fixture.err {
				require.NotNil(err)
				require.Contains(err.Error(), "Invalid value")
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestLogFormat(t *testing.T) {
	fixtures := []struct {
		format string
		err    bool
	}{{
		format: "text",
		err:    false,
	}, {
		format: "json",
		err:    false,
	}, {
		format: "txt",
		err:    true,
	}}

	app := setupLogCommand(t)

	for _, fixture := range fixtures {
		t.Run(fixture.format, func(t *testing.T) {
			require := require.New(t)

			err := app.Run([]string{"test", "nop",
				fmt.Sprintf("--log-format=%s", fixture.format)})

			if fixture.err {
				require.NotNil(err)
				require.Contains(err.Error(), "Invalid value")
			} else {
				require.NoError(err)
			}
		})
	}
}
