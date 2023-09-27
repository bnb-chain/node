package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, cdc *codec.Codec) {
	airdropCmd := &cobra.Command{
		Use:   "airdrop",
		Short: "airdrop commands",
	}

	airdropCmd.AddCommand(
		client.PostCommands(
			GetApprovalCmd(cdc),
		)...,
	)

	cmd.AddCommand(airdropCmd)
}
