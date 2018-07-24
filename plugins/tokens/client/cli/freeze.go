package commands

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
)

func freezeTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze",
		Short: "freeze some amount of token",
		RunE:  cmdr.freezeToken,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token to be frozen")
	cmd.Flags().StringP(flagAmount, "n", "", "amount of the token to be frozen")

	return cmd
}

func unfreezeTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfreeze",
		Short: "unfreeze some amount of token",
		RunE:  cmdr.unfreeze,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token to be frozen")
	cmd.Flags().StringP(flagAmount, "n", "", "amount of the token to be frozen")

	return cmd
}

func (c Commander) freezeToken(cmd *cobra.Command, args []string) error {
	freezeMsgBuilder := func(from sdk.AccAddress, symbol string, amount int64) sdk.Msg {
		return freeze.NewFreezeMsg(from, symbol, amount)
	}

	return c.checkAndSendTx(cmd, args, freezeMsgBuilder)
}

func (c Commander) unfreeze(cmd *cobra.Command, args []string) error {
	unfreezeMsgBuilder := func(from sdk.AccAddress, symbol string, amount int64) sdk.Msg {
		return freeze.NewUnfreezeMsg(from, symbol, amount)
	}

	return c.checkAndSendTx(cmd, args, unfreezeMsgBuilder)
}
