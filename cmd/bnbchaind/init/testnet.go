package init

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/app"
	appCfg "github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/wire"
)

var (
	flagNodeDirPrefix     = "node-dir-prefix"
	flagNumValidators     = "v"
	flagOutputDir         = "output-dir"
	flagNodeDaemonHome    = "node-daemon-home"
	flagNodeCliHome       = "node-cli-home"
	flagStartingIPAddress = "starting-ip-address"
	flagMonikers          = "monikers"
)

const nodeDirPerm = 0755

// get cmd to initialize all files for tendermint testnet and application
func TestnetFilesCmd(ctx *server.Context, cdc *wire.Codec, appInit server.AppInit) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "testnet",
		Short: "Initialize files for testnet",
		Long: `testnet will create "v" number of directories and populate each with
necessary files (private validator, genesis, config, etc.).

Note, strict routability for addresses is turned off in the config file.

Example:

	bnbchaind testnet --v 4 --output-dir ./output --starting-ip-address 192.168.10.2
	`,
		RunE: func(_ *cobra.Command, _ []string) error {
			config := ctx.Config
			return initTestnet(config, cdc, appInit)
		},
	}
	cmd.Flags().Int(flagNumValidators, 4,
		"Number of validators to initialize the testnet with",
	)
	cmd.Flags().StringP(flagOutputDir, "o", "./mytestnet",
		"Directory to store initialization data for the testnet",
	)
	cmd.Flags().String(flagNodeDirPrefix, "node",
		"Prefix the directory name for each node with (node results in node0, node1, ...)",
	)
	cmd.Flags().String(flagNodeDaemonHome, "gaiad",
		"Home directory of the node's daemon configuration",
	)
	cmd.Flags().String(flagNodeCliHome, "gaiacli",
		"Home directory of the node's cli configuration",
	)
	cmd.Flags().String(flagStartingIPAddress, "192.168.0.1",
		"Starting IP address (192.168.0.1 results in persistent peers list ID0@192.168.0.1:46656, ID1@192.168.0.2:46656, ...)")

	cmd.Flags().String(client.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")

	cmd.Flags().StringVar(&app.ServerContext.Bech32PrefixAccAddr, flagAccPrefix, "bnb", "bech32 prefix for AccAddress")
	app.ServerContext.BindPFlag("addr.bech32PrefixAccAddr", cmd.Flags().Lookup(flagAccPrefix))
	cmd.Flags().StringSlice(flagMonikers, nil, "specify monikers for nodes if needed")

	return cmd
}
func initTestnet(config *cfg.Config, cdc *codec.Codec, appInit server.AppInit) error {
	// func testnetWithConfig(config *cfg.Config, cdc *wire.Codec, appInit server.AppInit) error {
	var chainID string
	outDir := viper.GetString(flagOutputDir)
	numValidators := viper.GetInt(flagNumValidators)

	chainID = viper.GetString(client.FlagChainID)
	if chainID == "" {
		chainID = "chain-" + cmn.RandStr(6)
	}

	monikers := viper.GetStringSlice(flagMonikers)
	if len(monikers) != 0 && len(monikers) != numValidators {
		return fmt.Errorf("Len of monikers %d do not match validator num %d ", len(monikers), numValidators)
	}
	useCustomMoniker := true
	if len(monikers) == 0 {
		useCustomMoniker = false
		monikers = make([]string, numValidators)
	}
	nodeDirs := make([]string, numValidators)
	peers := make(map[string]string, numValidators) // moniker -> peer
	genTxsJson := make([]json.RawMessage, numValidators)
	genFiles := make([]string, numValidators)
	accs := make([]app.GenesisAccount, numValidators)

	// Generate private key, node ID, initial transaction
	for i := 0; i < numValidators; i++ {
		nodeDirName := fmt.Sprintf("%s%d", viper.GetString(flagNodeDirPrefix), i)
		nodeDir := filepath.Join(outDir, nodeDirName, viper.GetString(flagNodeDaemonHome))
		clientDir := filepath.Join(outDir, nodeDirName, viper.GetString(flagNodeCliHome))
		gentxsDir := filepath.Join(outDir, "gentxs")
		nodeDirs[i] = nodeDir

		config.SetRoot(nodeDir)
		cfg.EnsureRoot(config.RootDir)
		prepareClientDir(clientDir)
		if useCustomMoniker {
			config.Moniker = monikers[i]
		} else {
			monikers[i] = nodeDirName
			config.Moniker = nodeDirName
		}

		ip := getIP(i, viper.GetString(flagStartingIPAddress))
		nodeId, valPubKey := InitializeNodeValidatorFiles(config)

		addr, _ := createValOperAccount(clientDir, config.Moniker)
		nodeInfo := fmt.Sprintf("%s@%s:26656", nodeId, ip)
		peers[config.Moniker] = nodeInfo
		genTxsJson[i] = prepareGenTx(cdc, chainID, config.Moniker, nodeInfo, gentxsDir, addr, valPubKey)
		genFiles[i] = config.GenesisFile()
	}

	createGenesisFiles(cdc, chainID, genFiles, appInit, accs, genTxsJson)
	createConfigFiles(config, monikers, nodeDirs, peers)

	fmt.Printf("Successfully initialized %v node directories\n", numValidators)
	return nil
}

func prepareClientDir(clientDir string) {
	err := os.MkdirAll(clientDir, nodeDirPerm)
	if err != nil {
		panic(err)
	}
}

func prepareGenTx(cdc *codec.Codec, chainId, name, memo, gentxsDir string,
	valOperAddr sdk.ValAddress, valPubKey crypto.PubKey) json.RawMessage {
	txBytes := prepareCreateValidatorTx(cdc, chainId, name, memo, valOperAddr, valPubKey)
	err := writeFile(fmt.Sprintf("%v.json", name), gentxsDir, txBytes)
	if err != nil {
		panic(err)
	}
	return txBytes
}

func createGenesisFiles(cdc *codec.Codec, chainId string, genFiles []string, appInit server.AppInit, accs []app.GenesisAccount, genTxsJson []json.RawMessage) {
	genTime := utils.Now()
	appState, err := appInit.AppGenState(cdc, genTxsJson)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(genTxsJson); i++ {
		ExportGenesisFileWithTime(genFiles[i], chainId, nil, appState, genTime)
	}
}

func createConfigFiles(config *cfg.Config, monikers []string, nodeDirs []string, peers map[string]string) {
	numValidators := len(monikers)
	for i := 0; i < numValidators; i++ {
		config.Moniker = monikers[i]
		config.SetRoot(nodeDirs[i])

		var addressIps []string
		for moniker, peer := range peers {
			if monikers[i] != moniker {
				addressIps = append(addressIps, peer)
			}
		}
		sort.Strings(addressIps)
		persistentPeers := strings.Join(addressIps, ",")
		config.P2P.PersistentPeers = persistentPeers
		cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)

		appConfigFilePath := filepath.Join(config.RootDir, "config/", appCfg.AppConfigFileName+".toml")
		if _, err := os.Stat(appConfigFilePath); os.IsNotExist(err) {
			appCfg.WriteConfigFile(appConfigFilePath, app.ServerContext.BinanceChainConfig)
		}
	}
}

func getIP(i int, startingIPAddr string) string {
	if len(startingIPAddr) == 0 {
		if ip, err := server.ExternalIP(); err != nil {
			panic(err)
		} else {
			return ip
		}
	} else {
		if ip, err := calculateIP(startingIPAddr, i); err != nil {
			panic(err)
		} else {
			return ip
		}
	}
}

func calculateIP(ip string, i int) (string, error) {
	ipv4 := net.ParseIP(ip).To4()
	if ipv4 == nil {
		return "", fmt.Errorf("%v: non ipv4 address", ip)
	}

	for j := 0; j < i; j++ {
		ipv4[3]++
	}
	return ipv4.String(), nil
}

func writeFile(name string, dir string, contents []byte) error {
	writePath := filepath.Join(dir)
	file := filepath.Join(writePath, name)
	err := cmn.EnsureDir(writePath, 0700)
	if err != nil {
		return err
	}
	err = cmn.WriteFile(file, contents, 0600)
	if err != nil {
		return err
	}
	return nil
}
