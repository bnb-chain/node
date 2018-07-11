package commands

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/spf13/cobra"
)

const (
	flagSymbol = "symbol"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	dexCmd := &cobra.Command{
		Use:   "dex",
		Short: "dex commands",
	}

	dexCmd.AddCommand(
		client.PostCommands(
			listTradingPairCmd(cdc),
			client.LineBreak,
			makeOfferCmd(cdc),
			fillOfferCmd(cdc),
			cancelOfferCmd(cdc))...)
	// dexCmd.AddCommand(
	// 	client.GetCommands()...)

	dexCmd.AddCommand(client.LineBreak)
	cmd.AddCommand(dexCmd)
}
