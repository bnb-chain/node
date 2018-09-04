package commands

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/wire"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/wire"
)

const flagBaseAsset = "base-asset-symbol"
const flagQuoteAsset = "quote-asset-symbol"
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

			baseAsset := viper.GetString(flagBaseAsset)
			err = types.ValidateSymbol(baseAsset)
			if err != nil {
				return err
			}

			quoteAsset := viper.GetString(flagQuoteAsset)
			err = types.ValidateSymbol(quoteAsset)
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

			msg := list.NewMsg(from, baseAsset, quoteAsset, initPrice)
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagBaseAsset, "s", "", "symbol of the base asset")
	cmd.Flags().String(flagQuoteAsset, "", "symbol of the quote currency")
	cmd.Flags().String(flagInitPrice, "", "init price for this pair")

	return cmd
}
