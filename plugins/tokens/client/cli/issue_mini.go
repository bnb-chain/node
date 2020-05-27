package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/issue"
)

const (
	flagTokenUri = "token-uri"
)

func issueMiniTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue-mini",
		Short: "issue a new mini-token",
		RunE:  cmdr.issueMiniToken,
	}

	cmd.Flags().String(flagTokenName, "", "name of the new token")
	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the new token")
	cmd.Flags().Int64P(flagTotalSupply, "n", 0, "total supply of the new token")
	cmd.Flags().Bool(flagMintable, false, "whether the token can be minted")
	cmd.Flags().String(flagTokenUri, "", "uri of the token information")
	cmd.MarkFlagRequired(flagTotalSupply)
	return cmd
}

func (c Commander) issueMiniToken(cmd *cobra.Command, args []string) error {
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

	supply := viper.GetInt64(flagTotalSupply)
	err = checkMiniSupplyAmount(supply, int8(types.MiniRangeType))
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
	msg := issue.NewIssueMiniMsg(from, name, symbol, supply, mintable, tokenURI)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func checkMiniSupplyAmount(amount int64, tokenType int8) error {
	if amount <= types.MiniTokenMinExecutionAmount || amount > types.MiniTokenSupplyUpperBound {
		return errors.New("invalid supply amount")
	}
	if amount > types.SupplyRangeType(tokenType).UpperBound() {
		return errors.New(fmt.Sprintf("supply amount cannot exceed max supply amount of %s - %d", types.SupplyRangeType(tokenType).String(), types.SupplyRangeType(tokenType).UpperBound()))
	}
	return nil
}
