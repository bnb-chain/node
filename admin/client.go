package admin

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
			getModeCmd(cdc))...
		)

	adminCmd.AddCommand(client.LineBreak)
	cmd.AddCommand(adminCmd)
}

func setModeCmd(cdc *wire.Codec) *cobra.Command {
	cmd := cobra.Command{
		Use:   "set-mode [TransferOnly|RecoverOnly|Normal]",
		Short: "set the current running mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pvFile := viper.GetString(flagPVPath)
			privKey, _, err := readPrivValidator(pvFile)
			if err != nil {
				return err
			}

			mode := args[0]
			if strings.EqualFold(mode, "TransferOnly") ||
				strings.EqualFold(mode, "RecoverOnly") ||
				strings.EqualFold(mode, "Normal") {
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