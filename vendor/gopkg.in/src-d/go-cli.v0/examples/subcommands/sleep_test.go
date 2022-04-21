package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSleep(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on Windows")
	}

	fixtures := []struct {
		signal     os.Signal
		signalName string
	}{{
		signalName: "SIGTERM",
		signal:     syscall.SIGTERM,
	}, {
		signalName: "SIGINT",
		signal:     syscall.SIGINT,
	}}

	for _, fixture := range fixtures {
		t.Run(fixture.signalName, func(t *testing.T) {
			require := require.New(t)

			cmd := exec.Command(testBin, "sub", "sleep")

			stdout := bytes.NewBuffer(nil)
			stderr := bytes.NewBuffer(nil)
			cmd.Stdout = stdout
			cmd.Stderr = stderr

			ready := make(chan struct{})
			go func() {
				err := cmd.Run()
				require.NoError(err)
				ready <- struct{}{}
			}()

			if fixture.signal != nil {
				time.Sleep(1 * time.Second)
				p, oerr := os.FindProcess(cmd.Process.Pid)
				require.NoError(oerr)
				require.NoError(p.Signal(fixture.signal))
			}
			<-ready

			require.True(strings.Contains(stdout.String(), "Sleeping...\n"))
			if fixture.signal != nil {
				require.True(strings.Contains(stderr.String(),
					fmt.Sprintf("signal %s received", fixture.signalName)))
			}
		})
	}
}
