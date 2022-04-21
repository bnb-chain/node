package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var testBin string

func init() {
	bin := "test"
	if runtime.GOOS == "windows" {
		bin = "test.exe"
	}

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	testBin = filepath.Join(dir, bin)

	cmd := exec.Command("go", "build", "-o", testBin, ".")
	if err := cmd.Run(); err != nil {
		panic(err)
	}

}

func TestMain(t *testing.T) {
	require := require.New(t)

	cmd := exec.Command(testBin, "--help")

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	require.NoError(err)

	require.Empty(stderr.String())
	require.True(strings.HasPrefix(stdout.String(), "Usage:\n  basic"))
}
