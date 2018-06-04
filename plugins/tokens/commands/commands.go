package commands

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/spf13/cobra"
)

type Commander struct {
	Cdc *wire.Codec
}

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	tokencmd := &cobra.Command{
		Use:   "token",
		Short: "issue or view tokens",
		Long: ``,
	}

	cmdr := Commander{ Cdc: cdc }
	tokencmd.AddCommand(client.PostCommands(issueTokenCmd(cmdr))...)
	tokencmd.AddCommand(
		client.GetCommands(listTokensCmd,
		getTokenInfoCmd(cmdr))...)
	tokencmd.AddCommand(client.LineBreak)

	cmd.AddCommand(tokencmd)
}
