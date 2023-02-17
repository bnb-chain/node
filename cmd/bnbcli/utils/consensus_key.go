package utils

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/privval"
)

const (
	outputPathFlag = "output-path"
)

func genConsensusKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-consensus-key",
		Short: "generate the JSON file containing the private key to use as a validator in the consensus protocol",
		RunE:  runGenConsensusKeyCmd,
	}
	cmd.Flags().String(outputPathFlag, "./priv_validator_key.json", "The target path of the output file")
	return cmd
}

func runGenConsensusKeyCmd(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	outputPath, _ := flags.GetString(outputPathFlag)

	filePv := privval.GenFilePV(outputPath, "")
	filePv.Key.Save()
	fmt.Printf("The consensus key has been generated and saved to %s successfully\n", outputPath)
	pubkey := types.MustBech32ifyConsPub(filePv.Key.PubKey)
	fmt.Printf("The consensus pubkey is %s\n", pubkey)
	return nil
}
