package cli

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type NopCommand struct {
	Command `name:"nop" short-description:"nop" long-description:"nop"`
}

func (c *NopCommand) Execute(args []string) error {
	return nil
}

func setupDefaultCommand(t *testing.T) *App {
	app := New("test", "", "", "")
	app.AddCommand(&NopCommand{})
	return app
}

func TestProfilerOptions_Enable(t *testing.T) {
	require := require.New(t)
	app := setupDefaultCommand(t)
	err := app.Run([]string{"test", "nop", "--profiler-http", "--profiler-block-rate", "10"})
	require.NoError(err)
}

func TestProfilerOptions_Error(t *testing.T) {
	require := require.New(t)
	app := setupDefaultCommand(t)
	err := app.Run([]string{"test", "nop", "--profiler-http", "--profiler-endpoint", "a.b.c.d:foo"})
	require.Error(err)
}

func TestProfilerCPU(t *testing.T) {
	require := require.New(t)
	app := setupDefaultCommand(t)

	err := app.Run([]string{"test", "nop", "--profiler-cpu", "/directory/does/not/exist"})
	require.Error(err)

	tmp, err := ioutil.TempFile("", "cpu.prof")
	if err != nil {
		t.Fatalf("Could not create temporary file: %s", err.Error())
	}

	defer os.Remove(tmp.Name())

	err = app.Run([]string{"test", "nop", "--profiler-cpu", tmp.Name()})
	require.NoError(err)

	stat, err := tmp.Stat()
	require.NoError(err)
	require.NotZero(stat.Size())
}
