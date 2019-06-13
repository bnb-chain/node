package commands

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
	"github.com/binance-chain/node/common/scripts"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/account"
)

const (
	accountFlags = "account-flags"
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
			msg := account.NewSetAccountFlagsMsg(from, accountFlags)
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
				flags = appAccount.GetFlags()
			}
			flags = flags | scripts.TransferMemoCheckerFlag
			// build message
			msg := account.NewSetAccountFlagsMsg(from, flags)
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
				flags = appAccount.GetFlags()
			}
			invMemoCheck := ^uint64(scripts.TransferMemoCheckerFlag)
			flags = flags & invMemoCheck
			// build message
			msg := account.NewSetAccountFlagsMsg(from, flags)
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
