package main

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrint(t *testing.T) {
	fixtures := []struct {
		name   string
		args   []string
		stdout string
		stderr string
	}{{
		name:   "default",
		args:   []string{"print"},
		stdout: "Message: my-message\n",
	}, {
		name:   "one_message",
		args:   []string{"print", "--message", "hello"},
		stdout: "Message: hello\n",
	}}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			require := require.New(t)

			cmd := exec.Command(testBin, fixture.args...)

			stdout := bytes.NewBuffer(nil)
			stderr := bytes.NewBuffer(nil)
			cmd.Stdout = stdout
			cmd.Stderr = stderr

			err := cmd.Run()
			require.NoError(err)

			require.NoError(err)
			require.Equal(fixture.stderr, stderr.String())
			require.Equal(fixture.stdout, stdout.String())
		})
	}
}
