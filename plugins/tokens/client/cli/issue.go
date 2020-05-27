package commands

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/issue"
)

const (
	flagTotalSupply = "total-supply"
	flagTokenName   = "token-name"
	flagMintable    = "mintable"
)

func issueTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "issue a new token",
		RunE:  cmdr.issueToken,
	}

	cmd.Flags().String(flagTokenName, "", "name of the new token")
	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the new token")
	cmd.Flags().Int64P(flagTotalSupply, "n", 0, "total supply of the new token")
	cmd.Flags().Bool(flagMintable, false, "whether the token can be minted")
	cmd.MarkFlagRequired(flagTotalSupply)
	return cmd
}

func mintTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint",
		Short: "mint tokens for an existing token",
		RunE:  cmdr.mintToken,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token")
	cmd.Flags().Int64P(flagAmount, "n", 0, "amount to mint")
	cmd.MarkFlagRequired(flagAmount)
	return cmd
}

func (c Commander) issueToken(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)
	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	name := viper.GetString(flagTokenName)
	if len(name) == 0 {
		return errors.New("you must provide the name of the token")
	}

	symbol := viper.GetString(flagSymbol)
	err = types.ValidateIssueMsgTokenSymbol(symbol)
	if err != nil {
		return err
	}

	supply := viper.GetInt64(flagTotalSupply)
	err = checkSupplyAmount(supply)
	if err != nil {
		return err
	}

	mintable := viper.GetBool(flagMintable)

	// build message
	msg := issue.NewIssueMsg(from, name, symbol, supply, mintable)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func (c Commander) mintToken(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)
	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	symbol := viper.GetString(flagSymbol)
	amount := viper.GetInt64(flagAmount)

	if types.IsValidMiniTokenSymbol(strings.ToUpper(symbol)) {
		err = checkMiniTokenSupplyAmount(amount)
		if err != nil {
			return err
		}
	} else {
		err = types.ValidateMapperTokenSymbol(symbol)
		if err != nil {
			return err
		}
		err = checkSupplyAmount(amount)
		if err != nil {
			return err
		}
	}

	msg := issue.NewMintMsg(from, symbol, amount)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func checkSupplyAmount(amount int64) error {
	if amount <= 0 || amount > types.TokenMaxTotalSupply {
		return errors.New("invalid supply amount")
	}
	return nil
}
func checkMiniTokenSupplyAmount(amount int64) error {
	if amount <= types.MiniTokenMinExecutionAmount || amount > types.MiniTokenSupplyUpperBound {
		return errors.New("invalid supply amount")
	}

	return nil
}
