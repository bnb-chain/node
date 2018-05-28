package commands

import (
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func burnTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn",
		Short: "burn some amount of token",
		RunE:  cmdr.burnToken,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token to be burnt")
	cmd.Flags().StringP(flagAmount, "n", "", "amount of the token to be burnt")

	return cmd
}

func (c Commander) burnToken(cmd *cobra.Command, args []string) error {
	burnMsgBuilder := func(from sdk.Address, symbol string, amount int64) sdk.Msg {
		return burn.NewMsg(from, symbol, amount)
	}

	return c.checkAndSendTx(cmd, args, burnMsgBuilder)
}
