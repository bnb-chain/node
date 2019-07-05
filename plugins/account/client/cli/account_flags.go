package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	clientFlags "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/account/setaccountflags"
	"github.com/binance-chain/node/plugins/account/scripts"
	"github.com/binance-chain/node/wire"
)

const (
	accountFlags = "account-flags"
)

func setAccountFlagsCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-account-flags",
		Short: "set account flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txBldr := client.PrepareCtx(cdc)
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			flagsHexStr := viper.GetString(accountFlags)
			if !strings.HasPrefix(flagsHexStr, "0x") {
				return fmt.Errorf("flags must be hex string and start with 0x")
			}

			flagsHexStr = flagsHexStr[2:]
			accountFlags, err := strconv.ParseUint(flagsHexStr, 16, 64)
			if err != nil {
				return err
			}
			// build message
			msg := setaccountflags.NewSetAccountFlagsMsg(from, accountFlags)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}
	cmd.Flags().String(accountFlags, "", "account flags, hex encoding string with prefix 0x")
	return cmd
}

func enableMemoCheckFlagCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable-memo-checker",
		Short: "enable memo checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txBldr := client.PrepareCtx(cdc)
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			var flags uint64
			if viper.GetBool(clientFlags.FlagOffline) {
				flagsHexStr := viper.GetString(accountFlags)
				if len(flagsHexStr) == 0 {
					return fmt.Errorf("on offline mode, you must specify current account flags")
				}
				if !strings.HasPrefix(flagsHexStr, "0x") {
					return fmt.Errorf("flags must be hex string and start with 0x")
				}

				flagsHexStr = flagsHexStr[2:]
				flags, err = strconv.ParseUint(flagsHexStr, 16, 64)
				if err != nil {
					return err
				}
			} else {
				cliCtx := context.NewCLIContext().
					WithCodec(cdc).
					WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

				if err := cliCtx.EnsureAccountExistsFromAddr(from); err != nil {
					return err
				}
				acc, err := cliCtx.GetAccount(from)
				if err != nil {
					return err
				}
				appAccount, ok := acc.(types.NamedAccount)
				if !ok {
					return fmt.Errorf("unexpected account type")
				}
				flags = appAccount.GetFlags()
			}
			flags = flags | scripts.TransferMemoCheckerFlag
			// build message
			msg := setaccountflags.NewSetAccountFlagsMsg(from, flags)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}
	cmd.Flags().String(accountFlags, "", "account flags, hex encoding string with prefix 0x")
	return cmd
}

func disableMemoCheckFlagCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable-memo-checker",
		Short: "disable memo checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txBldr := client.PrepareCtx(cdc)
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			var flags uint64
			if viper.GetBool(clientFlags.FlagOffline) {
				flagsHexStr := viper.GetString(accountFlags)
				if len(flagsHexStr) == 0 {
					return fmt.Errorf("on offline mode, you must specify current account flags")
				}
				if !strings.HasPrefix(flagsHexStr, "0x") {
					return fmt.Errorf("flags must be hex string and start with 0x")
				}

				flagsHexStr = flagsHexStr[2:]
				flags, err = strconv.ParseUint(flagsHexStr, 16, 64)
				if err != nil {
					return err
				}
			} else {
				cliCtx := context.NewCLIContext().
					WithCodec(cdc).
					WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

				if err := cliCtx.EnsureAccountExistsFromAddr(from); err != nil {
					return err
				}
				acc, err := cliCtx.GetAccount(from)
				if err != nil {
					return err
				}
				appAccount, ok := acc.(types.NamedAccount)
				if !ok {
					return fmt.Errorf("unexpected account type")
				}
				flags = appAccount.GetFlags()
			}
			invMemoCheck := ^scripts.TransferMemoCheckerFlag
			flags = flags & invMemoCheck
			// build message
			msg := setaccountflags.NewSetAccountFlagsMsg(from, flags)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}
	cmd.Flags().String(accountFlags, "", "account flags, hex encoding string with prefix 0x")
	return cmd
}
