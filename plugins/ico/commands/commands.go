package commands

import (
	"github.com/spf13/cobra"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/client"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {
	cmd.AddCommand(
		client.PostCommands(issueTokenCmd(cdc))...
	)
}