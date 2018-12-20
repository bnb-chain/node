package admin

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/wire"
)

const (
	flagPVPath = "pvpath"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "admin commands",
	}

	adminCmd.AddCommand(
		client.GetCommands(
			setModeCmd(cdc),
			getModeCmd(cdc))...,
	)

	adminCmd.AddCommand(client.LineBreak)
	cmd.AddCommand(adminCmd)
}

func setModeCmd(cdc *wire.Codec) *cobra.Command {
	cmd := cobra.Command{
		Use:   "set-mode [0|1|2]",
		Short: "set the current running mode, 0: Normal, 1: TransferOnly, 2: RecoverOnly",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pvFile := viper.GetString(flagPVPath)
			privKey, _, err := readPrivValidator(pvFile)
			if err != nil {
				return err
			}

			mode := args[0]
			if mode == "0" || mode == "1" || mode == "2" {
				cliCtx := context.NewCLIContext().WithCodec(cdc)
				rand.Seed(time.Now().UnixNano())
				nonce := strconv.Itoa(rand.Int())
				sig, err := privKey.Sign([]byte(nonce))
				if err != nil {
					return err
				}
				res, err := cliCtx.QueryWithData(fmt.Sprintf("/admin/mode/%s/%s", mode, nonce), sig)
				if err != nil {
					return err
				}

				fmt.Println(res)
			} else {
				return errors.New("invalid mode")
			}

			return nil
		},
	}
	cmd.Flags().StringP(flagPVPath, "p", "", "path of priv_val file")
	return &cmd
}

func getModeCmd(cdc *wire.Codec) *cobra.Command {
	cmd := cobra.Command{
		Use:   "get-mode",
		Short: "get the current running mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			pvFile := viper.GetString(flagPVPath)
			privKey, _, err := readPrivValidator(pvFile)
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			rand.Seed(time.Now().UnixNano())
			nonce := strconv.Itoa(rand.Int())
			sig, err := privKey.Sign([]byte(nonce))
			if err != nil {
				return err
			}
			res, err := cliCtx.QueryWithData(fmt.Sprintf("/admin/mode/%s", nonce), sig)
			if err != nil {
				return err
			}
			fmt.Println(res)
			return nil
		},
	}
	cmd.Flags().StringP(flagPVPath, "p", "", "path of priv_val file")
	return &cmd
}
