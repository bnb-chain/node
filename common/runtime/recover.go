package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tendermint/tendermint/libs/common"

	"github.com/bnb-chain/node/common/log"
)

const fileName = "recover_params.json"

type runtimeParams struct {
	Mode Mode `json:"mode"`
}

func RecoverFromFile(homeDir string, defaultStartMode Mode) error {
	path := filepath.Join(homeDir, "config", fileName)
	var mode Mode
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Debug("path does not exist", "path", path)
		mode = defaultStartMode
	} else {
		params := mustReadFromFile(path)
		mode = params.Mode
	}

	return setRunningMode(mode)
}

func mustSaveToFile(path string, params *runtimeParams) {
	contents, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		panic(err)
	}
	err = common.WriteFileAtomic(path, contents, 0600)
	if err != nil {
		panic(err)
	}
}

func mustReadFromFile(path string) *runtimeParams {
	contents, err := common.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var res runtimeParams
	err = json.Unmarshal(contents, &res)
	if err != nil {
		panic(err)
	}
	return &res
}
