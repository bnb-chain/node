package init

/*
Why we overwrite the Init/Testnet functions in cosmos-sdk:
1. Cosmos moved init/testnet cmds to the gaia packages which we never and should not imports.
2. Cosmos has a different init/testnet workflow from ours. Also, the init cmd has some bugs.
3. After overwrite, the code is cleaner and easier to maintain.
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/BiJie/BinanceChain/app"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/wire"
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

// nolint: errcheck
func displayInfo(cdc *codec.Codec, info printInfo) error {
	out, err := codec.MarshalJSONIndent(cdc, info)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\n", string(out))
	return nil
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

			chainID := viper.GetString(client.FlagChainID)
			if chainID == "" {
				chainID = fmt.Sprintf("test-chain-%v", common.RandStr(6))
			}
			nodeID, pubKey := InitializeNodeValidatorFiles(config)

			config.Moniker = viper.GetString(flagMoniker)
			if config.Moniker == "" {
				config.Moniker = viper.GetString(client.FlagName)
			}
			if config.Moniker == "" {
				return errors.New("must specify --name (validator moniker)")
			}

			valOperAddr, secret := createValOperAccount(viper.GetString(flagClientHome), config.Moniker)
			memo := fmt.Sprintf("%s@%s:26656", nodeID, "127.0.0.1")
			genTx := prepareCreateValidatorTx(cdc, chainID, config.Moniker, memo, valOperAddr, pubKey)
			appState, err := appInit.AppGenState(cdc, []json.RawMessage{genTx})
			if err != nil {
				return err
			}
			genFile := config.GenesisFile()
			if !viper.GetBool(flagOverwrite) && common.FileExists(genFile) {
				return fmt.Errorf("genesis.json file already exists: %v", genFile)
			}
			ExportGenesisFileWithTime(genFile, chainID, nil, appState, utils.Now())
			writeConfigFile(config)

			toPrint := printInfo{
				ChainID:    chainID,
				Moniker:    config.Moniker,
				NodeID:     nodeID,
				AppMessage: makeAppMessage(cdc, secret),
			}
			return displayInfo(cdc, toPrint)
		},
	}

	cmd.Flags().StringP(flagClientHome, "c", app.DefaultCLIHome, "client's home directory")
	cmd.Flags().BoolP(flagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().String(client.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().String(flagMoniker, "", "overrides --name flag and set the validator's moniker to a different value; ignored if it runs without the --with-txs flag")
	cmd.MarkFlagRequired(flagMoniker)
	return cmd
}

func prepareCreateValidatorTx(cdc *codec.Codec, chainId, name, memo string,
	valOperAddr sdk.ValAddress, valPubKey crypto.PubKey) json.RawMessage {
	msg := stake.NewMsgCreateValidator(
		valOperAddr,
		valPubKey,
		app.DefaultSelfDelegationToken,
		stake.NewDescription(name, "", "", ""),
		stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
	)
	tx := auth.NewStdTx([]sdk.Msg{msg}, []auth.StdSignature{}, memo, auth.DefaultSource)
	txBldr := authtx.NewTxBuilderFromCLI().WithChainID(chainId).WithMemo(memo)
	signedTx, err := txBldr.SignStdTx(name, app.DefaultKeyPass, tx, false)
	if err != nil {
		panic(err)
	}

	txBytes, err := wire.MarshalJSONIndent(cdc, signedTx)
	if err != nil {
		panic(err)
	}

	return txBytes
}

func writeConfigFile(config *cfg.Config) {
	configFilePath := filepath.Join(config.RootDir, "config", "config.toml")
	cfg.WriteConfigFile(configFilePath, config)
}
