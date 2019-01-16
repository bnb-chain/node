package runtime

import (
	"fmt"
)

type Mode uint8

const (
	NormalMode Mode = iota
	TransferOnlyMode
	RecoverOnlyMode
)

var RunningMode = NormalMode

func SetRunningMode(mode Mode) error {
	if mode != NormalMode && mode != TransferOnlyMode && mode != RecoverOnlyMode {
		return fmt.Errorf("invalid mode %v", mode)
	}
	RunningMode = mode
	return nil
}
