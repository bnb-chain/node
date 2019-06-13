package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/validation"
	"github.com/binance-chain/node/plugins/tokens/account"
	"github.com/cosmos/cosmos-sdk/client/context"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
)

const (
	flags       = "flags"
	flagOffline = "offline"
)

func setAccountFlagsCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-account-flags",
		Short: "set account flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txBldr := client.PrepareCtx(cmdr.Cdc)
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			flagsHexStr := viper.GetString(flags)
			if !strings.HasPrefix(flagsHexStr, "0x") {
				return fmt.Errorf("flags must be hex string and start with 0x")
			}

			flagsHexStr = strings.ReplaceAll(flagsHexStr, "0x", "")
			accountFlags, err := strconv.ParseUint(flagsHexStr, 16, 64)
			if err != nil {
				return err
			}
			// build message
			msg := account.NewSetAccountFlagsMsg(from, accountFlags)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}
	cmd.Flags().String(flags, "", "account flags, hex encoding string")
	return cmd
}

func enableMemoCheckFlagCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable-memo-checker",
		Short: "enable memo checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txBldr := client.PrepareCtx(cmdr.Cdc)
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			var accountFlags uint64
			if viper.GetBool(flagOffline) {
				flagsHexStr := viper.GetString(flags)
				if len(flagsHexStr) == 0 {
					return fmt.Errorf("on offline mode, you must specify current account flags")
				}
				if !strings.HasPrefix(flagsHexStr, "0x") {
					return fmt.Errorf("flags must be hex string and start with 0x")
				}

				flagsHexStr = strings.ReplaceAll(flagsHexStr, "0x", "")
				accountFlags, err = strconv.ParseUint(flagsHexStr, 16, 64)
				if err != nil {
					return err
				}
			} else {
				cliCtx := context.NewCLIContext().
					WithCodec(cmdr.Cdc).
					WithAccountDecoder(authcmd.GetAccountDecoder(cmdr.Cdc))

				if err := cliCtx.EnsureAccountExistsFromAddr(from); err != nil {
					return err
				}
				acc, err := cliCtx.GetAccount(from)
				if err != nil {
					return err
				}
				appAccount := acc.(types.NamedAccount)
				accountFlags = appAccount.GetFlags()
			}
			accountFlags = accountFlags | validation.TransferMemoCheckerFlag
			// build message
			msg := account.NewSetAccountFlagsMsg(from, accountFlags)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}
	cmd.Flags().String(flags, "", "account flags, hex encoding string")
	return cmd
}

func disableMemoCheckFlagCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable-memo-checker",
		Short: "disable memo checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txBldr := client.PrepareCtx(cmdr.Cdc)
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			var accountFlags uint64
			if viper.GetBool(flagOffline) {
				flagsHexStr := viper.GetString(flags)
				if len(flagsHexStr) == 0 {
					return fmt.Errorf("on offline mode, you must specify current account flags")
				}
				if !strings.HasPrefix(flagsHexStr, "0x") {
					return fmt.Errorf("flags must be hex string and start with 0x")
				}

				flagsHexStr = strings.ReplaceAll(flagsHexStr, "0x", "")
				accountFlags, err = strconv.ParseUint(flagsHexStr, 16, 64)
				if err != nil {
					return err
				}
			} else {
				cliCtx := context.NewCLIContext().
					WithCodec(cmdr.Cdc).
					WithAccountDecoder(authcmd.GetAccountDecoder(cmdr.Cdc))

				if err := cliCtx.EnsureAccountExistsFromAddr(from); err != nil {
					return err
				}
				acc, err := cliCtx.GetAccount(from)
				if err != nil {
					return err
				}
				appAccount := acc.(types.NamedAccount)
				accountFlags = appAccount.GetFlags()
			}
			invMemoCheck := ^uint64(validation.TransferMemoCheckerFlag)
			accountFlags = accountFlags & invMemoCheck
			// build message
			msg := account.NewSetAccountFlagsMsg(from, accountFlags)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return client.SendOrPrintTx(cliCtx, txBldr, msg)
		},
	}
	cmd.Flags().String(flags, "", "account flags, hex encoding string")
	return cmd
}
