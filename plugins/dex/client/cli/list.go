package commands

import (
	"strings"

	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
)

const flagQuoteSymbol = "quote-symbol"
const flagInitPrice = "init-price"

func listTradingPairCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCoreContextFromViper().WithDecoder(types.GetAccountDecoder(cdc))

			from, err := ctx.GetFromAddress()
			if err != nil {
				return err
			}

			symbol := viper.GetString(flagSymbol)
			err = types.ValidateSymbol(symbol)
			if err != nil {
				return err
			}

			quoteSymbol := viper.GetString(flagQuoteSymbol)
			err = types.ValidateSymbol(quoteSymbol)
			if err != nil {
				return err
			}

			symbol = strings.ToUpper(symbol)
			quoteSymbol = strings.ToUpper(quoteSymbol)

			initPriceStr := viper.GetString(flagInitPrice)
			initPrice, err := utils.ParsePrice(initPriceStr)
			if err != nil {
				return err
			}

			msg := list.NewMsg(from, symbol, quoteSymbol, initPrice)
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the trading concurrency")
	cmd.Flags().String(flagQuoteSymbol, "", "symbol of the quote concurrency")
	cmd.Flags().String(flagInitPrice, "", "init price for this pair")

	return cmd
}
