package commands

import (
	"strings"

	"github.com/BiJie/BinanceChain/plugins/dex/order"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
)

const (
	flagId          = "id"
	flagPrice       = "price"
	flagQty         = "qty"
	flagSide        = "side"
	flagTimeInForce = "tif"
)

// NewOrderCommand -
func newOrderCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order -i <id> -l <pair> -s <side> -p <price> -q <qty> -t <timeInForce>",
		Short: "send new order",
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

			symbol = strings.ToUpper(symbol)
			id := viper.GetString(flagId)

			priceStr := viper.GetString(flagPrice)
			price, err := utils.ParsePrice(priceStr)
			if err != nil {
				return err
			}

			qtyStr := viper.GetString(flagPrice)
			qty, err := utils.ParsePrice(qtyStr)
			if err != nil {
				return err
			}

			tif := int8(viper.GetInt(flagTimeInForce))
			side := int8(viper.GetInt(flagSide))

			msg := order.NewNewOrderMsg(from, id, side, symbol, price, qty)
			msg.TimeInForce = tif
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringP(flagId, "i", "", "id string of the order")
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().StringP(flagSide, "s", "", "side (buy as 1 or sell as 2) of the order")
	cmd.Flags().StringP(flagPrice, "p", "", "price for the order")
	cmd.Flags().StringP(flagQty, "q", "", "quantity for the order")
	cmd.Flags().StringP(flagTimeInForce, "t", "", "TimeInForce for the order")
	return cmd
}

// CancelOrderCommand -
func showOrderBookCmd(cdc *wire.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "show [listed pair]",
		Short: "Show order book of the listed currency pair",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || len(args[0]) == 0 {
				return errors.New("You must provide a whatever")
			}
			return nil
		},
	}
}

// CancelOfferCmd -
func cancelOrderCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel -i <order id>",
		Short: "Cancel an order",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCoreContextFromViper().WithDecoder(types.GetAccountDecoder(cdc))

			from, err := ctx.GetFromAddress()
			if err != nil {
				return err
			}

			id := viper.GetString(flagId)

			msg := order.NewCancelOrderMsg(from, id)
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringP(flagId, "i", "", "id string of the order")
	return cmd
}
