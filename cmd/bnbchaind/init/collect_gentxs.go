package init

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagGenTxDir = "gentxs-dir"
	flagGenesisOutputFile = "genesis-output-file"
)

func CollectGenTxsCmd(cdc *codec.Codec, appInit server.AppInit) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect-gentxs --chain-id {chain-id} -i {genTxDir} -o {genesisOutputFile}",
		Short: "Collect genesis txs and output a genesis.json file",
		RunE: func(_ *cobra.Command, _ []string) error {
			genTxsDir := viper.GetString(flagGenTxDir)
			if genTxsDir == "" {
				return fmt.Errorf("%s must be provided", flagGenTxDir)
			}

			chainID := viper.GetString(client.FlagChainID)
			if chainID == "" {
				return fmt.Errorf("%s must be provided", client.FlagChainID)
			}

			output := viper.GetString(flagGenesisOutputFile)
			err := genGenesisFile(cdc, appInit, chainID, genTxsDir, output)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().String(client.FlagChainID, "", "genesis file chain-id, must be consistent with the chain-id used to sign the gentx")
	cmd.Flags().StringP(flagGenesisOutputFile, "o", ".genesis.json", "output genesis file")
	cmd.Flags().StringP(flagGenTxDir, "i", "",
		"override default \"gentx\" directory from which collect and execute "+
			"genesis transactions; default [--home]/config/gentx/")
	return cmd
}

func genGenesisFile(cdc *codec.Codec, appInit server.AppInit, chainId, genTxDir, genesisOutput string) (err error) {
	// process genesis transactions, else create default genesis.json
	genTxs, err := collectGenTxs(genTxDir, cdc)
	if err != nil {
		return
	}

	genTxsJson := make([]json.RawMessage, len(genTxs))
	for i, genTx := range genTxs {
		jsonRawTx, err := cdc.MarshalJSON(genTx)
		if err != nil {
			return err
		}
		genTxsJson[i] = jsonRawTx
	}

	appState, err := appInit.AppGenState(cdc, genTxsJson)
	if err != nil {
		return err
	}

	return ExportGenesisFile(genesisOutput, chainId, nil, appState)
}

func collectGenTxs(genTxsDir string, cdc *codec.Codec) (appGenTxs []auth.StdTx, err error) {
	var fos []os.FileInfo
	fos, err = ioutil.ReadDir(genTxsDir)
	if err != nil {
		return
	}

	for _, fo := range fos {
		filename := filepath.Join(genTxsDir, fo.Name())
		if !fo.IsDir() && (filepath.Ext(filename) != ".json") {
			continue
		}

		var jsonRawTx []byte
		if jsonRawTx, err = ioutil.ReadFile(filename); err != nil {
			return
		}
		var genStdTx auth.StdTx
		if err = cdc.UnmarshalJSON(jsonRawTx, &genStdTx); err != nil {
			return
		}
		appGenTxs = append(appGenTxs, genStdTx)
	}

	return
}
