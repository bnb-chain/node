package commands

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

const (
	flagId          = "id"
	flagRefId       = "refid"
	flagPrice       = "price"
	flagQty         = "qty"
	flagSide        = "side"
	flagTimeInForce = "tif"
)

func newOrderCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order -l <pair> -s <side> -p <price> -q <qty> -t <timeInForce>",
		Short: "Submit a new order",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCoreContextFromViper().WithDecoder(types.GetAccountDecoder(cdc))

			from, err := ctx.GetFromAddress()
			if err != nil {
				return err
			}

			symbol := viper.GetString(flagSymbol)
			err = validatePairSymbol(symbol)
			if err != nil {
				return err
			}

			symbol = strings.ToUpper(symbol)

			priceStr := viper.GetString(flagPrice)
			price, err := utils.ParsePrice(priceStr)
			if err != nil {
				return err
			}

			qtyStr := viper.GetString(flagQty)
			qty, err := utils.ParsePrice(qtyStr)
			if err != nil {
				return err
			}

			tif, err := order.TifStringToTifCode(viper.GetString(flagTimeInForce))
			if err != nil {
				panic(err)
			}
			side := int8(viper.GetInt(flagSide))

			msg, err := order.NewNewOrderMsgAuto(ctx, from, side, symbol, price, qty)
			if err != nil {
				panic(err)
			}

			msg.TimeInForce = tif
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}
			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().StringP(flagSide, "s", "", "side (buy as 1 or sell as 2) of the order")
	cmd.Flags().StringP(flagPrice, "p", "", "price for the order")
	cmd.Flags().StringP(flagQty, "q", "", "quantity for the order")
	cmd.Flags().StringP(flagTimeInForce, "t", "gtc", "TimeInForce for the order (gtc or ioc)")
	return cmd
}

func showOrderBookCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show -l <listed pair>",
		Short: "Show order book of the listed currency pair",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCoreContextFromViper().WithDecoder(types.GetAccountDecoder(cdc))

			symbol := viper.GetString(flagSymbol)
			err := validatePairSymbol(symbol)
			if err != nil {
				return err
			}

			ob, err := store.GetOrderBook(cdc, ctx, symbol)
			if err != nil {
				return err
			}
			levels := ob.Levels

			fmt.Printf("%16v|%16v|%16v|%16v\n", "SellQty", "SellPrice", "BuyPrice", "BuyQty")
			for _, l := range levels {
				fmt.Printf("%16v|%16v|%16v|%16v\n", l.SellQty, l.SellPrice, l.BuyPrice, l.BuyQty)
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	return cmd
}

func cancelOrderCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel -i <order id> -f <ref order id>",
		Short: "Cancel an order",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCoreContextFromViper().WithDecoder(types.GetAccountDecoder(cdc))

			from, err := ctx.GetFromAddress()
			if err != nil {
				return err
			}
			symbol := viper.GetString(flagSymbol)
			err = validatePairSymbol(symbol)
			if err != nil {
				return err
			}
			id := viper.GetString(flagId)
			if id == "" {
				return errors.New("please input order id")
			}
			refId := viper.GetString(flagRefId)
			if refId == "" {
				return errors.New("please input reference order id")
			}
			msg := order.NewCancelOrderMsg(from, symbol, id, refId)
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}
			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().StringP(flagId, "i", "", "id string of the cancellation")
	cmd.Flags().StringP(flagRefId, "f", "", "id string of the order")
	return cmd
}

func validatePairSymbol(symbol string) error {
	tokenSymbols := strings.Split(symbol, "_")
	if len(tokenSymbols) != 2 {
		return errors.New("Invalid symbol")
	}

	for _, tokenSymbol := range tokenSymbols {
		err := types.ValidateSymbol(tokenSymbol)
		if err != nil {
			return err
		}
	}

	return nil
}
