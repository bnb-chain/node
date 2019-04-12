package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/binance-chain/node/common/log"
	"github.com/tendermint/tendermint/config"
)

type Mode uint8

const (
	NormalMode Mode = iota
	TransferOnlyMode
	RecoverOnlyMode
)

var (
	runningMode = NormalMode
	mtx         = new(sync.RWMutex)
)

func GetRunningMode() Mode {
	mtx.RLock()
	defer mtx.RUnlock()
	return runningMode
}

func setRunningMode(mode Mode) error {
	if mode != NormalMode && mode != TransferOnlyMode && mode != RecoverOnlyMode {
		return fmt.Errorf("invalid mode %v", mode)
	}

	mtx.Lock()
	runningMode = mode
	mtx.Unlock()
	return nil
}

func UpdateRunningMode(cfg *config.Config, mode Mode) error {
	err := setRunningMode(mode)
	if err != nil {
		return err
	}
	var params *runtimeParams
	path := filepath.Join(cfg.RootDir, "config", fileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Debug("path does not exist", "path", path)
		params = &runtimeParams{Mode:mode}
	} else {
		params = mustReadFromFile(path)
		params.Mode = mode
	}
	mustSaveToFile(path, params)
	return nil
}
