package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/binance-chain/node/wire"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	dexCmd := &cobra.Command{
		Use:   "params",
		Short: "params commands",
	}
	dexCmd.AddCommand(
		client.PostCommands(
			SubmitFeeChangeProposalCmd(cdc))...)
	dexCmd.AddCommand(
		client.GetCommands(
			ShowFeeParamsCmd(cdc))...)
	cmd.AddCommand(dexCmd)
}
