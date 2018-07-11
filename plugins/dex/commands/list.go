package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
			initPrice, err := parseInitPrice(initPriceStr)
			if err != nil {
				return err
			}

			msg := list.NewMsg(from, symbol, quoteSymbol, initPrice)
			res, err := ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, msg, cdc)
			if err != nil {
				return err
			}

			fmt.Printf("Committed at block %d. Hash: %s\n", res.Height, res.Hash.String())
			return nil
		},
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the trading concurrency")
	cmd.Flags().String(flagQuoteSymbol, "", "symbol of the quote concurrency")
	cmd.Flags().String(flagInitPrice, "", "init price for this pair")

	return cmd
}

func parseInitPrice(initPriceStr string) (int64, error) {
	if len(initPriceStr) == 0 {
		return 0, errors.New("initPrice should be provided")
	}

	initPrice, err := strconv.ParseInt(initPriceStr, 10, 64)
	if err != nil {
		return 0, err
	}

	if initPrice <= 0 {
		return initPrice, errors.New("initPrice should be greater than 0")
	}

	return initPrice, nil
}
