package commands

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/client"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/wire"
)

const flagBaseAsset = "base-asset-symbol"
const flagQuoteAsset = "quote-asset-symbol"
const flagInitPrice = "init-price"
const flagProposalId = "proposal-id"

func listTradingPairCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txbldr := client.PrepareCtx(cdc)

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			baseAsset := viper.GetString(flagBaseAsset)
			err = types.ValidateMapperTokenSymbol(baseAsset)
			if err != nil {
				return err
			}

			quoteAsset := viper.GetString(flagQuoteAsset)
			err = types.ValidateMapperTokenSymbol(quoteAsset)
			if err != nil {
				return err
			}

			baseAsset = strings.ToUpper(baseAsset)
			quoteAsset = strings.ToUpper(quoteAsset)

			initPriceStr := viper.GetString(flagInitPrice)
			initPrice, err := utils.ParsePrice(initPriceStr)
			if err != nil {
				return err
			}

			proposalId := viper.GetInt64(flagProposalId)
			if proposalId <= 0 {
				return errors.New("proposal id should larger than zero")
			}

			msg := list.NewMsg(from, proposalId, baseAsset, quoteAsset, initPrice)
			err = client.SendOrPrintTx(cliCtx, txbldr, msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagBaseAsset, "s", "", "symbol of the base asset")
	cmd.Flags().String(flagQuoteAsset, "", "symbol of the quote currency")
	cmd.Flags().String(flagInitPrice, "", "init price for this pair")
	cmd.Flags().Int64(flagProposalId, 0, "list proposal id")

	return cmd
}
