package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, cdc *codec.Codec) {
	ownerShipCmd := &cobra.Command{
		Use:   "validator-ownership",
		Short: "validator-ownership commands",
		Long: `validator-ownership commands is a tool to help BSC validator operator create a mapping signature to New Validator on BSC
		# For example:
		bnbcli validator-ownership sign-validator-ownership \
		 --bsc-operator-address 0x45737bAf95D995a963ab3a7c9AC66fC7A63ad76E \
		 --from bsc-operator \
		 --chain-id Binance-Chain-Tigris`,
	}

	ownerShipCmd.AddCommand(
		client.PostCommands(
			SignValidatorOwnerShipCmd(cdc),
		)...,
	)

	cmd.AddCommand(ownerShipCmd)
}
