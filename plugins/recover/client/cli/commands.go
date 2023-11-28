package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, cdc *codec.Codec) {
	airdropCmd := &cobra.Command{
		Use:   "recover",
		Short: "recover commands",
	}

	airdropCmd.AddCommand(
		client.PostCommands(
			SignTokenRecoverRequestCmd(cdc),
		)...,
	)

	cmd.AddCommand(airdropCmd)
}
