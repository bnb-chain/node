package cli

import (
	"errors"

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
	flagSequence        = "channel-sequence"
	flagContractAddress = "contract-address"
	flagSenderAddress   = "sender-address"
	flagReceiverAddress = "receiver-address"
	flagAmount          = "amount"
	flagRelayFee        = "relay-fee"
)

// TransferCmd implements cross chain transfer.
func TransferCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer token to receiver",
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
			println(receiverAddressStr)
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

			msg := types.NewTransferMsg(sequence,
				types.NewEthereumAddress(contractAddress),
				types.NewEthereumAddress(senderAddress),
				receiverAddr,
				amountToTransfer,
				relayFee,
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

	cmd.Flags().Int64(flagSequence, 0, "sequence of transfer channel")
	cmd.Flags().String(flagContractAddress, "", "contract address")
	cmd.Flags().String(flagSenderAddress, "", "sender address")
	cmd.Flags().String(flagReceiverAddress, "", "receiver address")
	cmd.Flags().String(flagAmount, "", "amount of transfer token")
	cmd.Flags().String(flagRelayFee, "", "amount of relay fee")

	return cmd
}

func TimeoutCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeout",
		Short: "Transfer timeout",
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
