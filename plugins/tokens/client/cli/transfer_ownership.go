package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/common/client"
	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/plugins/tokens/ownership"
)

const flagNewOwner = "new-owner"

func transferOwnershipCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-ownership",
		Short: "transfer the ownership of token",
		RunE:  cmdr.transferOwnership,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token to be transferred owner")
	cmd.Flags().StringP(flagNewOwner, "", "", "new owner of the token")

	return cmd
}

func (c Commander) transferOwnership(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)
	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	symbol := viper.GetString(flagSymbol)
	if !types.IsValidMiniTokenSymbol(symbol) {
		err = types.ValidateTokenSymbol(symbol)
		if err != nil {
			return err
		}
	}
	symbol = strings.ToUpper(symbol)

	newOwnerStr := viper.GetString(flagNewOwner)
	if len(newOwnerStr) == 0 {
		return fmt.Errorf("newOwner can not be empty")
	}
	newOwner, err := sdk.AccAddressFromBech32(newOwnerStr)
	if err != nil {
		return err
	}

	msg := ownership.NewTransferOwnershipMsg(from, symbol, newOwner)

	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}
