package admin

import (
	"errors"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/privval"
)

// read of create the private key file for this config
func readPrivValidator(privValKeyFile string) (crypto.PrivKey, crypto.PubKey, error) {
	var privValidator *privval.FilePV

	if common.FileExists(privValKeyFile) {
		privValidator = privval.LoadFilePVEmptyState(privValKeyFile, "")
	} else {
		return nil, nil, errors.New("priv_val file does not exist")
	}

	return privValidator.Key.PrivKey, privValidator.GetPubKey(), nil
}
