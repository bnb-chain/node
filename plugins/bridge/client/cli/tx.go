package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/oracle"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/plugins/bridge/types"
	"github.com/binance-chain/node/wire"
)

const (
	flagSequence         = "channel-sequence"
	flagSequenceToSkip   = "sequence-to-skip"
	flagClaimTypeToSkip  = "claim-type-to-skip"
	flagContractAddress  = "contract-address"
	flagRefundAddress    = "refund-address"
	flagRecipientAddress = "recipient-address"
	flagAmount           = "amount"
	flagSymbol           = "symbol"
	flagRelayFee         = "relay-fee"
	flagContractDecimals = "contract-decimals"
	flagToAddress        = "to"
	flagBindStatus       = "bind-status"
	flagExpireTime       = "expire-time"
	flagRefundReason     = "refund-reason"

	flagChannelId = "channel-id"
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
			refundAddressStr := viper.GetString(flagRefundAddress)
			recipientAddressStr := viper.GetString(flagRecipientAddress)
			amountStr := viper.GetString(flagAmount)
			relayFeeStr := viper.GetString(flagRelayFee)
			expireTime := viper.GetInt64(flagExpireTime)
			symbol := viper.GetString(flagSymbol)

			if sequence < 0 {
				return errors.New("sequence should not be less than 0")
			}

			if contractAddress == "" {
				return errors.New("contract address should not be empty")
			}

			if relayFeeStr == "" {
				return errors.New("relay fee should not be empty")
			}

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			relayFee, err := sdk.ParseCoin(relayFeeStr)
			if err != nil {
				return err
			}

			var refundAddressList []types.EthereumAddress
			var recipientAddressList []sdk.AccAddress
			var transferAmountList []int64
			refundAddressStrList := strings.Split(refundAddressStr, ",")
			recipientAddressStrList := strings.Split(recipientAddressStr, ",")
			amountToTransferStrList := strings.Split(amountStr, ",")
			if len(refundAddressStrList) != len(recipientAddressStrList) || len(refundAddressStrList) != len(amountToTransferStrList) {
				return fmt.Errorf("the length of refund address array, recipient address array and transfer amount array must be the same")
			}
			for _, str := range refundAddressStrList {
				refundAddressList = append(refundAddressList, types.NewEthereumAddress(str))
			}
			for _, str := range recipientAddressStrList {
				addr, err := sdk.AccAddressFromBech32(str)
				if err != nil {
					return err
				}
				recipientAddressList = append(recipientAddressList, addr)
			}
			for _, str := range amountToTransferStrList {
				amount, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					return err
				}
				transferAmountList = append(transferAmountList, amount)
			}

			claim := types.TransferInClaim{
				ContractAddress:   types.NewEthereumAddress(contractAddress),
				RefundAddresses:   refundAddressList,
				ReceiverAddresses: recipientAddressList,
				Amounts:           transferAmountList,
				Symbol:            symbol,
				RelayFee:          relayFee,
				ExpireTime:        expireTime,
			}
			claimBz, err := json.Marshal(claim)
			if err != nil {
				return err
			}
			msg := oracle.NewClaimMsg(types.ClaimTypeTransferIn, sequence, string(claimBz), fromAddr)
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
	cmd.Flags().String(flagRefundAddress, "", "array of refund address")
	cmd.Flags().String(flagRecipientAddress, "", "array of recipient address")
	cmd.Flags().String(flagAmount, "", "array of transfer amount")
	cmd.Flags().String(flagSymbol, "", "symbol")
	cmd.Flags().String(flagRelayFee, "", "amount of relay fee, example: 10000:BNB")
	cmd.Flags().Int64(flagExpireTime, 0, "expire timestamp(s)")

	return cmd
}

func UpdateTransferOutCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-transfer-out",
		Short: "update transfer out",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			sequence := viper.GetInt64(flagSequence)
			amount := viper.GetString(flagAmount)
			refundReasonStr := viper.GetString(flagRefundReason)
			if len(refundReasonStr) == 0 {
				return fmt.Errorf("empty refund reason")
			}
			refundReason := types.ParseRefundReason(refundReasonStr)

			refundAddr, err := sdk.AccAddressFromBech32(viper.GetString(flagRefundAddress))
			if err != nil {
				return err
			}

			if sequence < 0 {
				return errors.New("sequence should be no less than 0")
			}

			if amount == "" {
				return errors.New("amount should not be empty")
			}

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			amountToTransfer, err := sdk.ParseCoin(amount)
			if err != nil {
				return err
			}

			claim := types.UpdateTransferOutClaim{
				RefundAddress: refundAddr,
				Amount:        amountToTransfer,
				RefundReason:  refundReason,
			}
			claimBz, err := json.Marshal(claim)
			if err != nil {
				return err
			}

			msg := oracle.NewClaimMsg(types.ClaimTypeUpdateTransferOut, sequence, string(claimBz), fromAddr)

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
	cmd.Flags().String(flagRefundAddress, "", "sender address")
	cmd.Flags().String(flagAmount, "", "amount of transfer token")
	cmd.Flags().String(flagRefundReason, "", "refund reason: unboundToken, timeout, insufficientBalance")

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
			contractAddress := viper.GetString(flagContractAddress)
			symbol := viper.GetString(flagSymbol)
			status := types.ParseBindStatus(viper.GetString(flagBindStatus))

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}
			amount, ok := sdk.NewIntFromString(viper.GetString(flagAmount))
			if !ok {
				return fmt.Errorf("invalid amount")
			}

			claim := types.UpdateBindClaim{
				Status:          status,
				Symbol:          strings.ToUpper(symbol),
				Amount:          amount,
				ContractAddress: types.NewEthereumAddress(contractAddress),
			}

			claimBz, err := json.Marshal(claim)
			if err != nil {
				return err
			}

			claimMsg := oracle.NewClaimMsg(types.ClaimTypeUpdateBind, sequence, string(claimBz), fromAddr)
			err = claimMsg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{claimMsg})
			}

			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{claimMsg})
		},
	}

	cmd.Flags().Int64(flagSequence, 0, "sequence of transfer channel")
	cmd.Flags().String(flagContractAddress, "", "contract address")
	cmd.Flags().Int(flagContractDecimals, 0, "contract token decimals")
	cmd.Flags().String(flagAmount, "", "amount to bind")
	cmd.Flags().String(flagSymbol, "", "symbol")
	cmd.Flags().String(flagBindStatus, "", "bind status: success, timeout, rejected, invalidParameter")

	return cmd
}

func QueryProphecy(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-prophecy",
		Short: "query prophecy",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			sequence := viper.GetInt64(flagSequence)
			channelId := viper.GetInt(flagChannelId)

			key := oracle.GetClaimId(sdk.ClaimType(channelId), sequence)
			res, err := cliCtx.QueryStore([]byte(key), common.OracleStoreName)
			if err != nil {
				return err
			}

			if len(res) == 0 {
				fmt.Printf("No such claim exists\n")
				return nil
			}

			dbProphecy := new(oracle.DBProphecy)
			err = cdc.UnmarshalBinaryBare(res, &dbProphecy)
			if err != nil {
				return err
			}

			prophecy, err := dbProphecy.DeserializeFromDB()
			if err != nil {
				return err
			}

			output, err := wire.MarshalJSONIndent(cdc, prophecy)
			if err != nil {
				return err
			}
			fmt.Println(string(output))

			return nil
		},
	}

	cmd.Flags().Int64(flagSequence, 0, "sequence of channel")
	cmd.Flags().Int(flagChannelId, 0, "channel id")

	return cmd
}

func SkipSequenceCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skip-sequence",
		Short: "skip sequence",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			sequence := viper.GetInt64(flagSequence)
			sequenceToSkip := viper.GetInt64(flagSequenceToSkip)
			claimTypeToSkip := viper.GetInt(flagClaimTypeToSkip)

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			claim := types.SkipSequenceClaim{
				ClaimType: types.ClaimTypeUpdateBind,
				Sequence:  sequenceToSkip,
			}

			claimBz, err := json.Marshal(claim)
			if err != nil {
				return err
			}

			claimMsg := oracle.NewClaimMsg(sdk.ClaimType(claimTypeToSkip), sequence, string(claimBz), fromAddr)

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{claimMsg})
			}

			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{claimMsg})
		},
	}

	cmd.Flags().Int64(flagSequence, 0, "sequence of skip claim type")
	cmd.Flags().Int64(flagSequenceToSkip, 00, "sequence to skip")

	return cmd
}
