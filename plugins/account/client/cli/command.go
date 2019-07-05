package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/binance-chain/node/wire"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	scriptsCmd := &cobra.Command{
		Use:   "script",
		Short: "set account flags for customized script",
	}

	scriptsCmd.AddCommand(
		client.PostCommands(
			setAccountFlagsCmd(cdc),
			enableMemoCheckFlagCmd(cdc),
			disableMemoCheckFlagCmd(cdc))...)
	cmd.AddCommand(scriptsCmd)
}
