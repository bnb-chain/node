package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, cdc *codec.Codec) {
	bridgeCmd := &cobra.Command{
		Use:   "bridge",
		Short: "bridge commands",
	}

	bridgeCmd.AddCommand(
		client.PostCommands(
			TransferInCmd(cdc),
			UpdateTransferOutCmd(cdc),
			BindCmd(cdc),
			TransferOutCmd(cdc),
			UpdateBindCmd(cdc),
		)...,
	)

	bridgeCmd.AddCommand(client.LineBreak)

	bridgeCmd.AddCommand(
		client.GetCommands(
			QueryProphecy(cdc))...,
	)
	cmd.AddCommand(bridgeCmd)
}
