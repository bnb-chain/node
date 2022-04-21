package cli

import (
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	require := require.New(t)
	app := New("test", "0.1.0", "abcde", "test app")
	var (
		stdout, stderr string
		err            error
	)

	stdout = capturer.CaptureStdout(func() {
		stderr = capturer.CaptureStderr(func() {
			err = app.Run([]string{"test", "version"})
		})
	})

	require.NoError(err)
	require.Empty(stderr)
	require.Equal("test version 0.1.0 build abcde\n", stdout)
}
