package commands

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/binance-chain/node/wire"
)

const (
	flagSymbol = "symbol"
	flagAmount = "amount"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	miniTokenCmd := &cobra.Command{
		Use:   "miniToken",
		Short: "issue or view mini tokens",
		Long:  ``,
	}

	cmdr := Commander{Cdc: cdc}
	miniTokenCmd.AddCommand(
		client.PostCommands(
			issueMiniTokenCmd(cmdr),
			mintMiniTokenCmd(cmdr))...)

	miniTokenCmd.AddCommand(client.LineBreak)

	cmd.AddCommand(miniTokenCmd)
}
