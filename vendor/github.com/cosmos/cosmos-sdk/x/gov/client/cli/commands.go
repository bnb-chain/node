package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

const (
	storeGov = "gov"
)

func AddCommands(cmd *cobra.Command, cdc *codec.Codec) {
	govCmd := &cobra.Command{
		Use:   "gov",
		Short: "gov commands",
	}

	govCmd.AddCommand(
		client.PostCommands(
			GetCmdDeposit(cdc),
			GetCmdSubmitProposal(cdc),
			GetCmdSubmitListProposal(cdc),
			GetCmdSubmitDelistProposal(cdc),
			GetCmdVote(cdc),
		)...,
	)

	govCmd.AddCommand(client.LineBreak)

	govCmd.AddCommand(
		client.GetCommands(
			GetCmdQueryProposal(storeGov, cdc),
			GetCmdQueryProposals(storeGov, cdc),
			GetCmdQueryDeposit(storeGov, cdc),
			GetCmdQueryDeposits(storeGov, cdc),
			GetCmdQueryVote(storeGov, cdc),
			GetCmdQueryVotes(storeGov, cdc),
		)...,
	)
	cmd.AddCommand(govCmd)
}
