package cli

import (
	"errors"
	"fmt"

	"github.com/binance-chain/node/common/client"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/plugins/bridge/types"
)

const (
	flagSequence         = "channel-sequence"
	flagContractAddress  = "contract-address"
	flagSenderAddress    = "sender-address"
	flagReceiverAddress  = "receiver-address"
	flagAmount           = "amount"
	flagSymbol           = "symbol"
	flagRelayFee         = "relay-fee"
	flagContractDecimals = "contract-decimals"
	flagToAddress        = "to"
	flagStatus           = "status"
	flagExpireTime       = "expire-time"
)

func TransferInCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-in",
		Short: "transfer smart chain token to binance chain receiver",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			sequence := viper.GetInt64(flagSequence)
			contractAddress := viper.GetString(flagContractAddress)
			senderAddress := viper.GetString(flagSenderAddress)
			receiverAddressStr := viper.GetString(flagReceiverAddress)
			amount := viper.GetString(flagAmount)
			relayFeeStr := viper.GetString(flagRelayFee)
			expireTime := viper.GetInt64(flagExpireTime)

			if sequence <= 0 {
				return errors.New("sequence should not be less than 0")
			}

			if contractAddress == "" {
				return errors.New("contract address should not be empty")
			}

			if senderAddress == "" {
				return errors.New("sender address should not be empty")
			}

			if receiverAddressStr == "" {
				return errors.New("receiver address should not be empty")
			}

			if amount == "" {
				return errors.New("amount should not be empty")
			}

			if relayFeeStr == "" {
				return errors.New("relay fee should not be empty")
			}

			receiverAddr, err := sdk.AccAddressFromBech32(receiverAddressStr)
			if err != nil {
				println(err.Error())
				return err
			}

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			amountToTransfer, err := sdk.ParseCoin(amount)
			if err != nil {
				return err
			}

			relayFee, err := sdk.ParseCoin(relayFeeStr)
			if err != nil {
				return err
			}

			msg := types.NewTransferInMsg(sequence,
				types.NewEthereumAddress(contractAddress),
				types.NewEthereumAddress(senderAddress),
				receiverAddr,
				amountToTransfer,
				relayFee,
				fromAddr,
				expireTime,
			)

			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().Int64(flagSequence, 0, "sequence of transfer channel")
	cmd.Flags().String(flagContractAddress, "", "contract address")
	cmd.Flags().String(flagSenderAddress, "", "sender address")
	cmd.Flags().String(flagReceiverAddress, "", "receiver address")
	cmd.Flags().String(flagAmount, "", "amount of transfer token")
	cmd.Flags().String(flagRelayFee, "", "amount of relay fee")
	cmd.Flags().Int64(flagExpireTime, 0, "expire period, counted by second")

	return cmd
}

func TransferOutTimeoutCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-out-timeout",
		Short: "Transfer out timeout",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			sequence := viper.GetInt64(flagSequence)
			senderAddressStr := viper.GetString(flagSenderAddress)
			amount := viper.GetString(flagAmount)

			if sequence <= 0 {
				return errors.New("sequence should not be less than 0")
			}

			if senderAddressStr == "" {
				return errors.New("sender address should not be empty")
			}

			if amount == "" {
				return errors.New("amount should not be empty")
			}

			senderAddr, err := sdk.AccAddressFromBech32(viper.GetString(senderAddressStr))
			if err != nil {
				return err
			}

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			amountToTransfer, err := sdk.ParseCoin(amount)
			if err != nil {
				return err
			}

			msg := types.NewTimeoutMsg(senderAddr, sequence,
				amountToTransfer,
				fromAddr,
			)

			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().Int64(flagSequence, 0, "sequence of timeout channel")
	cmd.Flags().String(flagSenderAddress, "", "sender address")
	cmd.Flags().String(flagAmount, "", "amount of transfer token")

	return cmd
}

func BindCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind",
		Short: "bind smart chain token to bep2 token",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			contractAddress := viper.GetString(flagContractAddress)
			contractDecimals := viper.GetInt(flagContractDecimals)
			amount := viper.GetInt64(flagAmount)
			symbol := viper.GetString(flagSymbol)
			expireTime := viper.GetInt64(flagExpireTime)

			// build message
			msg := types.NewBindMsg(from, symbol, amount, types.NewEthereumAddress(contractAddress), int8(contractDecimals), expireTime)

			sdkErr := msg.ValidateBasic()
			if sdkErr != nil {
				return fmt.Errorf("%v", sdkErr.Data())
			}
			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}

	cmd.Flags().String(flagContractAddress, "", "contract address")
	cmd.Flags().Int(flagContractDecimals, 0, "contract token decimals")
	cmd.Flags().Int64(flagAmount, 0, "amount to bind")
	cmd.Flags().String(flagSymbol, "", "symbol")
	cmd.Flags().Int64(flagExpireTime, 0, "expire timestamp(s)")

	return cmd
}

func TransferOutCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-out",
		Short: "transfer bep2 token to smart chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			to := viper.GetString(flagToAddress)
			amount := viper.GetString(flagAmount)
			expireTime := viper.GetInt64(flagExpireTime)

			amountToTransfer, err := sdk.ParseCoin(amount)
			if err != nil {
				return err
			}

			// build message
			msg := types.NewTransferOutMsg(from, types.NewEthereumAddress(to), amountToTransfer, expireTime)

			sdkErr := msg.ValidateBasic()
			if sdkErr != nil {
				return fmt.Errorf("%v", sdkErr.Data())
			}
			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}

	cmd.Flags().String(flagToAddress, "", "smart chain address")
	cmd.Flags().String(flagAmount, "", "amount")
	cmd.Flags().Int64(flagExpireTime, 0, "expire timestamp(s)")

	return cmd
}

func UpdateBindCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-bind",
		Short: "update bind",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			sequence := viper.GetInt64(flagSequence)
<<<<<<< HEAD
=======
			symbol := viper.GetString(flagSymbol)
>>>>>>> fix bug in update bind handler
			contractAddress := viper.GetString(flagContractAddress)
			contractDecimals := viper.GetInt(flagContractDecimals)
			amount := viper.GetInt64(flagAmount)
			symbol := viper.GetString(flagSymbol)
			status := viper.GetInt(flagStatus)

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}
			amount, ok := sdk.NewIntFromString(viper.GetString(flagAmount))
			if !ok {
				return fmt.Errorf("invalid amount")
			}

			msg := types.NewUpdateBindMsg(sequence,
				fromAddr,
				symbol,
				amount,
				types.NewEthereumAddress(contractAddress),
				int8(contractDecimals),
				types.BindStatus(status),
			)

			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

<<<<<<< HEAD
	cmd.Flags().Int64(flagSequence, 0, "sequence of transfer channel")
	cmd.Flags().String(flagContractAddress, "", "contract address")
	cmd.Flags().Int(flagContractDecimals, 0, "contract token decimals")
	cmd.Flags().Int64(flagAmount, 0, "amount to bind")
	cmd.Flags().String(flagSymbol, "", "symbol")
	cmd.Flags().Int(flagStatus, 0, "status")
=======
	cmd.Flags().String(flagAmount, "", "bind amount")
	cmd.Flags().String(flagContractAddress, "", "ERC20 contract address")
	cmd.Flags().Int(flagContractDecimals, 0, "ERC20 token decimals")
	cmd.Flags().String(flagSymbol, "", "bep2 token symbol")
	cmd.Flags().Int64(flagSequence, 0, "sequence of bind status channel")
	cmd.Flags().String(flagStatus, "", "bind status")
>>>>>>> fix bug in update bind handler

	return cmd
}
