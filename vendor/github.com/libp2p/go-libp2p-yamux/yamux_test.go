package sm_yamux

import (
	"testing"

	tmux "github.com/libp2p/go-libp2p-testing/suites/mux"
)

func TestYamuxTransport(t *testing.T) {
	tmux.SubtestAll(t, DefaultTransport)
}
