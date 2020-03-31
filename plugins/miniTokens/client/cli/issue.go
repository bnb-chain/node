package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/minitokens/issue"
)

const (
	flagMaxTotalSupply = "max-total-supply"
	flagTotalSupply = "total-supply"
	flagTokenName   = "token-name"
	flagMintable    = "mintable"
	flagTokenUri    = "token-uri"
)

func issueMiniTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "issue a new mini-token",
		RunE:  cmdr.issueToken,
	}

	cmd.Flags().String(flagTokenName, "", "name of the new token")
	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the new token")
	cmd.Flags().Int64P(flagMaxTotalSupply, "m", 0, "max total supply of the new token")
	cmd.Flags().Int64P(flagTotalSupply, "n", 0, "total supply of the new token")
	cmd.Flags().Bool(flagMintable, false, "whether the token can be minted")
	cmd.Flags().String(flagTokenUri, "", "uri of the token information")
	cmd.MarkFlagRequired(flagMaxTotalSupply)
	cmd.MarkFlagRequired(flagTotalSupply)
	return cmd
}

func mintMiniTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint",
		Short: "mint mini tokens for an existing token",
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
	err = types.ValidateIssueMsgMiniTokenSymbol(symbol)
	if err != nil {
		return err
	}

	maxSupply := viper.GetInt64(flagMaxTotalSupply)
	err = checkMaxSupplyAmount(maxSupply)
	if err != nil {
		return err
	}

	supply := viper.GetInt64(flagTotalSupply)
	err = checkSupplyAmount(supply, maxSupply)
	if err != nil {
		return err
	}

	mintable := viper.GetBool(flagMintable)

	tokenURI := viper.GetString(flagTokenUri)
	err = validateTokenURI(tokenURI)
	if err != nil {
		return err
	}

	// build message
	msg := issue.NewIssueMsg(from, name, symbol, maxSupply, supply, mintable, tokenURI)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func (c Commander) mintToken(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)
	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	symbol := viper.GetString(flagSymbol)
	err = types.ValidateMapperMiniTokenSymbol(symbol)
	if err != nil {
		return err
	}

	amount := viper.GetInt64(flagAmount)
	err = checkSupplyAmount(amount, 0)
	if err != nil {
		return err
	}

	msg := issue.NewMintMsg(from, symbol, amount)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func checkMaxSupplyAmount(amount int64) error {
	if amount <= types.MiniTokenMinTotalSupply || amount > types.MiniTokenMaxTotalSupplyUpperBound {
		return errors.New("invalid max supply amount")
	}
	return nil
}

func checkSupplyAmount(amount, maxAmount int64) error {
	if amount <= types.MiniTokenMinTotalSupply || amount > types.MiniTokenMaxTotalSupplyUpperBound {
		return errors.New("invalid supply amount")
	}
	if maxAmount > 0 && amount > maxAmount {
		return errors.New("supply amount cannot exceed max supply amount")
	}
	return nil
}
