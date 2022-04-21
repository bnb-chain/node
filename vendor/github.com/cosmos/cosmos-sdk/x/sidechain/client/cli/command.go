package cli

import (
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"

	"github.com/cosmos/cosmos-sdk/client"
)

const (
	flagTitle        = "title"
	flagDeposit      = "deposit"
	flagVotingPeriod = "voting-period"
	flagSideChainId  = "side-chain-id"
)

func AddCommands(cmd *cobra.Command, cdc *amino.Codec) {

	dexCmd := &cobra.Command{
		Use:   "side-chain",
		Short: "side chain management commands",
	}
	dexCmd.AddCommand(
		client.PostCommands(
			SubmitChannelManageProposalCmd(cdc))...)
	dexCmd.AddCommand(
		client.GetCommands(
			ShowChannelPermissionCmd(cdc))...)
	cmd.AddCommand(dexCmd)
}
