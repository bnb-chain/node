package commands

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	cli "github.com/cosmos/cosmos-sdk/client/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authctx "github.com/cosmos/cosmos-sdk/x/auth/client/context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/wire"
)

const flagQuoteSymbol = "quote-symbol"
const flagInitPrice = "init-price"

func listTradingPairCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			txCtx := authctx.NewTxContextFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().WithAccountDecoder(types.GetAccountDecoder(cdc))

			from, err := cliCtx.GetFromAddress()
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
			err = cli.SendTx(txCtx, cliCtx, []sdk.Msg{msg})
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
