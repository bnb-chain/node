package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"
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
	flagSideChainId      = "side-chain-id"
	flagContractAddress  = "contract-address"
	flagAmount           = "amount"
	flagSymbol           = "symbol"
	flagContractDecimals = "contract-decimals"
	flagToAddress        = "to"
	flagExpireTime       = "expire-time"

	flagChannelId = "channel-id"
)

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
			msg := types.NewBindMsg(from, symbol, amount, types.NewSmartChainAddress(contractAddress), int8(contractDecimals), expireTime)

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

func UnbindCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbind",
		Short: "unbind smart chain token to bep2 token",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			symbol := viper.GetString(flagSymbol)

			// build message
			msg := types.NewUnbindMsg(from, symbol)

			sdkErr := msg.ValidateBasic()
			if sdkErr != nil {
				return fmt.Errorf("%v", sdkErr.Data())
			}
			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}

	cmd.Flags().String(flagSymbol, "", "symbol")
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
			msg := types.NewTransferOutMsg(from, types.NewSmartChainAddress(to), amountToTransfer, expireTime)

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

func QueryProphecy(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-prophecy",
		Short: "query oracle prophecy",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			sequence := viper.GetInt64(flagSequence)
			chainId := viper.GetUint(flagSideChainId)
			channelId := viper.GetUint(flagChannelId)

			key := oracle.GetClaimId(sdk.ChainID(chainId), sdk.ChannelID(channelId), uint64(sequence))
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
	cmd.Flags().Uint16(flagSideChainId, 0, "side chain id")

	return cmd
}
