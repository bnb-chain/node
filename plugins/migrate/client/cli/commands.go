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
	}

	ownerShipCmd.AddCommand(
		client.PostCommands(
			SignValidatorOwnerShipCmd(cdc),
		)...,
	)

	cmd.AddCommand(ownerShipCmd)
}
