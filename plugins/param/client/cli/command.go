package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/binance-chain/node/wire"
)

const (
	flagTitle        = "title"
	flagDescription  = "description"
	flagDeposit      = "deposit"
	flagVotingPeriod = "voting-period"
	flagSideChainId  = "side-chain-id"
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
		client.PostCommands(
			SubmitCSCParamChangeProposalCmd(cdc))...)
	dexCmd.AddCommand(
		client.PostCommands(
			SubmitSCParamChangeProposalCmd(cdc))...)
	dexCmd.AddCommand(
		client.GetCommands(
			ShowFeeParamsCmd(cdc))...)
	cmd.AddCommand(dexCmd)
}
