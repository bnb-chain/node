package init

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"

	"github.com/bnb-chain/node/app"
)

// ExportGenesisFile creates and writes the genesis configuration to disk. An
// error is returned if building or writing the configuration to file fails.
func ExportGenesisFile(
	genFile, chainID string, validators []types.GenesisValidator, appState json.RawMessage,
) error {

	genDoc := types.GenesisDoc{
		ChainID:    chainID,
		Validators: validators,
		AppState:   appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genFile)
}

// ExportGenesisFileWithTime creates and writes the genesis configuration to disk.
// An error is returned if building or writing the configuration to file fails.
func ExportGenesisFileWithTime(
	genFile, chainID string, validators []types.GenesisValidator,
	appState json.RawMessage, genTime time.Time,
) error {

	genDoc := types.GenesisDoc{
		GenesisTime: genTime,
		ChainID:     chainID,
		Validators:  validators,
		AppState:    appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genFile)
}

// read of create the private key file for this config
func ReadOrCreatePrivValidator(privValKeyFile, privValStateFile string) crypto.PubKey {
	var privValidator *privval.FilePV

	if common.FileExists(privValKeyFile) && common.FileExists(privValStateFile) {
		privValidator = privval.LoadFilePV(privValKeyFile, privValStateFile)
	} else {
		privValidator = privval.GenFilePV(privValKeyFile, privValStateFile)
		privValidator.Save()
	}

	return privValidator.GetPubKey()
}

// InitializeNodeValidatorFiles creates private validator and p2p configuration files.
func InitializeNodeValidatorFiles(config *cfg.Config) (nodeID string, valPubKey crypto.PubKey) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		panic(err)
	}

	nodeID = string(nodeKey.ID())
	valPubKey = ReadOrCreatePrivValidator(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())

	return nodeID, valPubKey
}

func CreateValOperAccount(clientDir, keyName string) (sdk.ValAddress, string) {
	accAddr, secret, err := server.GenerateSaveCoinKey(clientDir, keyName, app.DefaultKeyPass, true)
	if err != nil {
		panic(err)
	}

	info := map[string]string{"secret": secret}

	keySeed, err := json.Marshal(info)
	if err != nil {
		panic(err)
	}

	// save private key seed words
	err = writeFile(fmt.Sprintf("%v.json", "key_seed"), clientDir, keySeed)
	if err != nil {
		panic(err)
	}

	return sdk.ValAddress(accAddr.Bytes()), secret
}

func makeAppMessage(cdc *codec.Codec, secret string) json.RawMessage {
	mm := map[string]string{"secret": secret}
	bz, err := cdc.MarshalJSON(mm)
	if err != nil {
		panic(err)
	}

	return json.RawMessage(bz)
}
