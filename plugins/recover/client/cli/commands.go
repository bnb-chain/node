package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, cdc *codec.Codec) {
	recoverCmd := &cobra.Command{
		Use:   "recover",
		Short: "recover commands",
		Long: `recover commands is a tool for users to sign a request to recover their tokens from Binance Chain to Binance Smart Chain
		# For example:
		bnbcli recover sign-token-recover-request \
		 --amount 19999999000000000 \
		 --token-symbol BNB \
		 --recipient 0x5b38da6a701c568545dcfcb03fcb875f56beddc4 \
		 --from user1 \
		 --chain-id Binance-Chain-Tigris
		`,
	}

	recoverCmd.AddCommand(
		client.PostCommands(
			SignTokenRecoverRequestCmd(cdc),
		)...,
	)

	cmd.AddCommand(recoverCmd)
}
