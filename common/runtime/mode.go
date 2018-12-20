package runtime

type Mode uint8

const (
	NormalMode Mode = iota
	TransferOnlyMode
	RecoverOnlyMode
)

var RunningMode = NormalMode

func SetRunningMode(mode Mode) {
	RunningMode = mode
}

