package commands

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	clientflag "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	txutils "github.com/cosmos/cosmos-sdk/client/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"

	"github.com/bnb-chain/node/common/client"
	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/common/utils"
	"github.com/bnb-chain/node/plugins/dex"
	"github.com/bnb-chain/node/plugins/dex/order"
	"github.com/bnb-chain/node/plugins/dex/store"
	"github.com/bnb-chain/node/wire"
)

const (
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
			cliCtx, txBldr := client.PrepareCtx(cdc)
			from, err := cliCtx.GetFromAddress()
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
				return err
			}
			side := int8(viper.GetInt(flagSide))

			// avoids an ugly panin sequence 0 with --dry
			if viper.GetBool(clientflag.FlagOffline) {
				txBldr = txBldr.WithSequence(viper.GetInt64(clientflag.FlagSequence))
			} else {
				acc, err := cliCtx.GetAccount(from)
				if acc == nil || err != nil {
					fmt.Println("No transactions involving this address yet. Using sequence 0.")
					txBldr = txBldr.WithSequence(0)
				} else {
					err = client.EnsureSequence(cliCtx, &txBldr)
					if err != nil {
						return err
					}
				}
			}

			msg, err := order.NewNewOrderMsgAuto(txBldr, from, side, symbol, price, qty)
			if err != nil {
				return err
			}

			msg.TimeInForce = tif

			err = client.SendOrPrintTx(cliCtx, txBldr, msg)
			if err != nil {
				return err
			}

			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().IntP(flagLevels, "L", 100, "maximum level (1,5,10,20,50,100,500,1000) to return")
	cmd.Flags().StringP(flagSide, "s", "", "side (buy as 1 or sell as 2) of the order")
	cmd.Flags().StringP(flagPrice, "p", "", "price for the order")
	cmd.Flags().StringP(flagQty, "q", "", "quantity for the order")
	cmd.Flags().StringP(flagTimeInForce, "t", "gte", "TimeInForce for the order (gte or ioc)")
	return cmd
}

func showOrderBookCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show -l <listed pair>",
		Short: "Show order book of the listed currency pair",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCLIContext().WithAccountDecoder(types.GetAccountDecoder(cdc))

			symbol := viper.GetString(flagSymbol)
			err := validatePairSymbol(symbol)
			if err != nil {
				return err
			}
			levelsLimit := viper.GetInt(flagLevels)
			if levelsLimit <= 0 || levelsLimit > dex.MaxDepthLevels {
				return fmt.Errorf("%s should be greater than 0 and not exceed %d", flagLevels, dex.MaxDepthLevels)
			}

			ob, err := store.GetOrderBook(cdc, ctx, symbol, levelsLimit)
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

	cmd.Flags().IntP(flagLevels, "L", 100, "maximum level (1,5,10,20,50,100,500,1000) to return")
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	return cmd
}

func cancelOrderCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel -l <trading pair> -f <ref order id>",
		Short: "Cancel an order",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := txbuilder.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().WithCodec(cdc).WithAccountDecoder(types.GetAccountDecoder(cdc))
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}
			symbol := viper.GetString(flagSymbol)
			err = validatePairSymbol(symbol)
			if err != nil {
				return err
			}
			refId := viper.GetString(flagRefId)
			if refId == "" {
				return errors.New("please input reference order id")
			}
			msg := order.NewCancelOrderMsg(from, symbol, refId)
			if cliCtx.GenerateOnly {
				return txutils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			err = txutils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
			if err != nil {
				return err
			}
			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().StringP(flagRefId, "f", "", "id string of the order")
	return cmd
}

func validatePairSymbol(symbol string) error {
	return store.ValidatePairSymbol(symbol)
}
