package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/BiJie/BinanceChain/wire"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	dexCmd := &cobra.Command{
		Use:   "params",
		Short: "params commands",
	}
	dexCmd.AddCommand(
		client.PostCommands(
			GetCmdSubmitFeeChangeProposal(cdc))...)

	cmd.AddCommand(dexCmd)
}
