package utils

import (
	"encoding/json"
	"fmt"
	"github.com/BiJie/BinanceChain/app"
	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/client"
	serverCfg "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/tendermint/tendermint/crypto"
	cmn "github.com/tendermint/tendermint/libs/common"
	pvm "github.com/tendermint/tendermint/privval"
	tmtypes "github.com/tendermint/tendermint/types"
	"io/ioutil"
	"path"
	"sort"

	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/p2p"
)

const (
	flagWithTxs      = "with-txs"
	flagOverwrite    = "overwrite"
	flagClientHome   = "home-client"
	flagOverwriteKey = "overwrite-key"
	flagSkipGenesis  = "skip-genesis"
	flagMoniker      = "moniker"
	FlagGenTxs       = "gen-txs"
)

type initConfig struct {
	ChainID   string
	GenTxs    bool
	GenTxsDir string
	Overwrite bool
}

type printInfo struct {
	Moniker    string          `json:"moniker"`
	ChainID    string          `json:"chain_id"`
	NodeID     string          `json:"node_id"`
	AppMessage json.RawMessage `json:"app_message"`
}

// genesis piece structure for creating combined genesis
type GenesisTx struct {
	NodeID    string                   `json:"node_id"`
	IP        string                   `json:"ip"`
	Validator tmtypes.GenesisValidator `json:"validator"`
	AppGenTx  json.RawMessage          `json:"app_gen_tx"`
}

// nolint: errcheck
func displayInfo(cdc *codec.Codec, info printInfo) error {
	out, err := codec.MarshalJSONIndent(cdc, info)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\n", string(out))
	return nil
}

func gentxWithConfig(cdc *wire.Codec, appInit server.AppInit, config *cfg.Config, genTxConfig serverCfg.GenTx) (
	cliPrint json.RawMessage, genTxFile json.RawMessage, err error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return
	}
	nodeID := string(nodeKey.ID())
	pubKey := readOrCreatePrivValidator(config)

	appGenTx, cliPrint, validator, err := appInit.AppGenTx(cdc, pubKey, genTxConfig)
	if err != nil {
		return
	}

	tx := GenesisTx{
		NodeID:    nodeID,
		IP:        genTxConfig.IP,
		Validator: validator,
		AppGenTx:  appGenTx,
	}
	bz, err := wire.MarshalJSONIndent(cdc, tx)
	if err != nil {
		return
	}
	genTxFile = json.RawMessage(bz)
	name := fmt.Sprintf("gentx-%v.json", nodeID)
	writePath := filepath.Join(config.RootDir, "config", "gentx")
	file := filepath.Join(writePath, name)
	err = cmn.EnsureDir(writePath, 0700)
	if err != nil {
		return
	}
	err = cmn.WriteFile(file, bz, 0644)
	if err != nil {
		return
	}

	// Write updated config with moniker
	config.Moniker = genTxConfig.Name
	configFilePath := filepath.Join(config.RootDir, "config", "config.toml")
	cfg.WriteConfigFile(configFilePath, config)

	return
}

// get cmd to initialize all files for tendermint and application
// nolint
func InitCmd(ctx *server.Context, cdc *codec.Codec, appInit server.AppInit) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize private validator, p2p, genesis, and application configuration files",
		Long: `Initialize validators's and node's configuration files.

Note that only node's configuration files will be written if the flag --skip-genesis is
enabled, and the genesis file will not be generated.
`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))

			name := viper.GetString(client.FlagName)
			chainID := viper.GetString(client.FlagChainID)
			if chainID == "" {
				chainID = fmt.Sprintf("test-chain-%v", common.RandStr(6))
			}
			nodeID, _, err := InitializeNodeValidatorFiles(config)
			if err != nil {
				return err
			}

			if viper.GetString(flagMoniker) != "" {
				config.Moniker = viper.GetString(flagMoniker)
			}
			if config.Moniker == "" && name != "" {
				config.Moniker = name
			}
			toPrint := printInfo{
				ChainID: chainID,
				Moniker: config.Moniker,
				NodeID:  nodeID,
			}
			if viper.GetBool(flagSkipGenesis) {
				cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)
				return displayInfo(cdc, toPrint)
			}

			initCfg := initConfig{
				ChainID:   chainID,
				GenTxs:    viper.GetBool(FlagGenTxs),
				GenTxsDir: filepath.Join(config.RootDir, "config", "gentx"),
				Overwrite: viper.GetBool(flagOverwrite),
			}
			_, _, toPrint.AppMessage, err = initWithConfig(cdc, appInit, config, initCfg)
			// print out some key information
			if err != nil {
				return err
			}
			return displayInfo(cdc, toPrint)
		},
	}

	// TODO(#118): need to figure out why cosmos add this flag: https://github.com/cosmos/cosmos-sdk/blob/v0.23.1/server/init.go
	cmd.Flags().String(cli.HomeFlag, app.DefaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(flagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().String(client.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().Bool(flagWithTxs, false, "apply existing genesis transactions from [--home]/config/gentx/")
	cmd.Flags().String(client.FlagName, "", "name of private key with which to sign the gentx")
	cmd.Flags().String(flagMoniker, "", "overrides --name flag and set the validator's moniker to a different value; ignored if it runs without the --with-txs flag")
	// TODO(#118): need to figure out why cosmos add this flag: https://github.com/cosmos/cosmos-sdk/blob/v0.23.1/server/init.go
	cmd.Flags().String(flagClientHome, app.DefaultCLIHome, "client's home directory")
	cmd.Flags().Bool(flagOverwriteKey, false, "overwrite client's key")
	cmd.Flags().Bool(flagSkipGenesis, false, "do not create genesis.json")
	return cmd
}

// InitializeNodeValidatorFiles creates private validator and p2p configuration files.
func InitializeNodeValidatorFiles(config *cfg.Config) (nodeID string, valPubKey crypto.PubKey, err error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return
	}
	nodeID = string(nodeKey.ID())
	valPubKey = readOrCreatePrivValidator(config)
	return
}

func initWithConfig(cdc *codec.Codec, appInit server.AppInit, config *cfg.Config, initConfig initConfig) (
	chainID string, nodeID string, appMessage json.RawMessage, err error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return
	}
	nodeID = string(nodeKey.ID())
	pubKey := readOrCreatePrivValidator(config)

	if initConfig.ChainID == "" {
		initConfig.ChainID = fmt.Sprintf("test-chain-%v", cmn.RandStr(6))
	}
	chainID = initConfig.ChainID

	genFile := config.GenesisFile()
	if !initConfig.Overwrite && cmn.FileExists(genFile) {
		err = fmt.Errorf("genesis.json file already exists: %v", genFile)
		return
	}

	// process genesis transactions, or otherwise create one for defaults
	var appGenTxs []json.RawMessage
	var validators []tmtypes.GenesisValidator
	var persistentPeers string

	if initConfig.GenTxs {
		validators, appGenTxs, persistentPeers, err = processGenTxs(initConfig.GenTxsDir, cdc)
		if err != nil {
			return
		}
		config.P2P.PersistentPeers = persistentPeers
		configFilePath := filepath.Join(config.RootDir, "config", "config.toml")
		cfg.WriteConfigFile(configFilePath, config)
	} else {
		genTxConfig := serverCfg.GenTx{
			Name:    viper.GetString(client.FlagName),
			CliRoot: viper.GetString(cli.HomeFlag),
			IP:      "127.0.0.1",
		}

		// Write updated config with moniker
		config.Moniker = genTxConfig.Name
		configFilePath := filepath.Join(config.RootDir, "config", "config.toml")
		cfg.WriteConfigFile(configFilePath, config)
		appGenTx, am, validator, err := appInit.AppGenTx(cdc, pubKey, genTxConfig)
		appMessage = am
		if err != nil {
			return "", "", nil, err
		}
		validators = []tmtypes.GenesisValidator{validator}
		appGenTxs = []json.RawMessage{appGenTx}
	}

	appState, err := appInit.AppGenState(cdc, appGenTxs)
	if err != nil {
		return
	}

	err = writeGenesisFile(cdc, genFile, initConfig.ChainID, validators, appState)
	if err != nil {
		return
	}

	return
}

// append a genesis-piece
func processGenTxs(genTxsDir string, cdc *wire.Codec) (
	validators []tmtypes.GenesisValidator, appGenTxs []json.RawMessage, persistentPeers string, err error) {

	var fos []os.FileInfo
	fos, err = ioutil.ReadDir(genTxsDir)
	if err != nil {
		return
	}

	genTxs := make(map[string]GenesisTx)
	var nodeIDs []string
	for _, fo := range fos {
		filename := path.Join(genTxsDir, fo.Name())
		if !fo.IsDir() && (path.Ext(filename) != ".json") {
			continue
		}

		// get the genTx
		var bz []byte
		bz, err = ioutil.ReadFile(filename)
		if err != nil {
			return
		}
		var genTx GenesisTx
		err = cdc.UnmarshalJSON(bz, &genTx)
		if err != nil {
			return
		}

		genTxs[genTx.NodeID] = genTx
		nodeIDs = append(nodeIDs, genTx.NodeID)
	}

	sort.Strings(nodeIDs)

	for _, nodeID := range nodeIDs {
		genTx := genTxs[nodeID]

		// combine some stuff
		validators = append(validators, genTx.Validator)
		appGenTxs = append(appGenTxs, genTx.AppGenTx)

		// Add a persistent peer
		comma := ","
		if len(persistentPeers) == 0 {
			comma = ""
		}
		persistentPeers += fmt.Sprintf("%s%s@%s:26656", comma, genTx.NodeID, genTx.IP)
	}

	return
}

// writeGenesisFile creates and writes the genesis configuration to disk. An
// error is returned if building or writing the configuration to file fails.
func writeGenesisFile(cdc *wire.Codec, genesisFile, chainID string, validators []tmtypes.GenesisValidator, appState json.RawMessage) error {
	genDoc := tmtypes.GenesisDoc{
		ChainID:    chainID,
		Validators: validators,
		AppState:   appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genesisFile)
}

// read of create the private key file for this config
func readOrCreatePrivValidator(tmConfig *cfg.Config) crypto.PubKey {
	// private validator
	privValFile := tmConfig.PrivValidatorFile()
	var privValidator *pvm.FilePV
	if cmn.FileExists(privValFile) {
		privValidator = pvm.LoadFilePV(privValFile)
	} else {
		privValidator = pvm.GenFilePV(privValFile)
		privValidator.Save()
	}
	return privValidator.GetPubKey()
}
