package commands

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
)

const (
	flagId          = "id"
	flagRefId       = "refid"
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
			err = validatePairSymbol(symbol)
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

			msg := order.NewNewOrderMsg(from, id, side, symbol, price, qty)
			msg.TimeInForce = tif
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}
			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagId, "i", "", "id string of the order")
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().StringP(flagSide, "s", "", "side (buy as 1 or sell as 2) of the order")
	cmd.Flags().StringP(flagPrice, "p", "", "price for the order")
	cmd.Flags().StringP(flagQty, "q", "", "quantity for the order")
	cmd.Flags().StringP(flagTimeInForce, "t", "gtc", "TimeInForce for the order (gtc or ioc)")
	return cmd
}

// CancelOrderCommand -
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

			bz, err := ctx.Query(fmt.Sprintf("app/orderbook/%s", symbol))
			if err != nil {
				return err
			}

			orderbook := make([][]int64, 10)
			err = cdc.UnmarshalBinary(bz, &orderbook)
			if err != nil {
				return err
			}

			fmt.Printf("%16v|%16v|%16v|%16v\n", "SellQty", "SellPrice", "BuyPrice", "BuyQty")
			for _, l := range orderbook {
				fmt.Printf("%16v|%16v|%16v|%16v\n", l[0], l[1], l[2], l[3])
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	return cmd
}

// CancelOfferCmd -
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

			id := viper.GetString(flagId)
			if id == "" {
				fmt.Println("please input order id")
			}
			refId := viper.GetString(flagRefId)
			if refId == "" {
				fmt.Println("please input reference order id")
			}
			msg := order.NewCancelOrderMsg(from, id, refId)
			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}
			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagId, "i", "", "id string of the order")
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
