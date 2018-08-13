package commands

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/BiJie/BinanceChain/wire"
)

const (
	flagSymbol = "symbol"
	flagAmount = "amount"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "issue or view tokens",
		Long:  ``,
	}

	cmdr := Commander{Cdc: cdc}
	tokenCmd.AddCommand(
		client.PostCommands(
			issueTokenCmd(cmdr),
			burnTokenCmd(cmdr),
			freezeTokenCmd(cmdr),
			unfreezeTokenCmd(cmdr))...)
	tokenCmd.AddCommand(
		client.GetCommands(
			listTokensCmd,
			getTokenInfoCmd(cmdr))...)

	tokenCmd.AddCommand(client.LineBreak)
	cmd.AddCommand(tokenCmd)
}
