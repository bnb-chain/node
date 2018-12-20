package admin

import (
	"errors"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/privval"
)

// read of create the private key file for this config
func readPrivValidator(privValFile string) (crypto.PrivKey, crypto.PubKey, error) {
	var privValidator *privval.FilePV

	if common.FileExists(privValFile) {
		privValidator = privval.LoadFilePV(privValFile)
	} else {
		return nil, nil, errors.New("priv_val file does not exist")
	}

	return privValidator.PrivKey, privValidator.PubKey, nil
}
