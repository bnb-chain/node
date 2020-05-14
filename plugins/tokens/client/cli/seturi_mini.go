package commands

import (
	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/seturi_mini"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func setTokenURICmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-uri-mini --symbol {symbol} --uri {token uri} --from {token issuer address}",
		Short: "set token URI of mini-token",
		RunE:  cmdr.setTokenURI,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the mini-token")
	cmd.Flags().String(flagTokenUri, "", "uri of the token information")

	return cmd
}

func (c Commander) setTokenURI(cmd *cobra.Command, args []string) error {
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
	tokenURI := viper.GetString(flagTokenUri)
	err = validateTokenURI(tokenURI)
	if err != nil {
		return err
	}

	msg := seturi_mini.NewSetUriMsg(from, symbol, tokenURI)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}
