package commands

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/plugins/tokens/timelock"

	"github.com/binance-chain/node/common/client"
)

const (
	flagLockTime    = "lock-time"
	flagDescription = "description"
	flagTimeLockId  = "time-lock-id"
	flagAccount     = "account"
)

func timeLockCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time-lock",
		Short: "time lock tokens",
		RunE:  cmdr.timeLock,
	}

	cmd.Flags().String(flagAmount, "", "amount of tokens to lock")
	cmd.Flags().Int64(flagLockTime, 0, "timestamp of lock time(second)")
	cmd.Flags().String(flagDescription, "", "description of time lock")

	return cmd
}

func (c Commander) timeLock(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)
	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	description := viper.GetString(flagDescription)
	if len(description) == 0 {
		return fmt.Errorf("description should not be empty")
	}

	if len(description) > timelock.MaxDescriptionLength {
		return fmt.Errorf("length of description should be less than %d", timelock.MaxDescriptionLength)
	}

	amount, err := sdk.ParseCoins(viper.GetString(flagAmount))
	if err != nil {
		return err
	}

	lockTime := viper.GetInt64(flagLockTime)
	if lockTime <= 0 {
		return fmt.Errorf("lock time should be positive")
	}

	if time.Unix(lockTime, 0).Before(time.Now()) {
		return fmt.Errorf("lock time(%s) should be after now", time.Unix(lockTime, 0).UTC().String())
	}

	// build message
	msg := timelock.NewTimeLockMsg(from, description, amount, lockTime)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func timeUnlockCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time-unlock",
		Short: "time unlock tokens",
		RunE:  cmdr.timeUnlock,
	}

	cmd.Flags().String(flagTimeLockId, "", "time lock id")

	return cmd
}

func (c Commander) timeUnlock(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	timeLockId := viper.GetInt64(flagTimeLockId)
	if timeLockId < timelock.InitialRecordId {
		return fmt.Errorf("lock time should not less than %d", timelock.InitialRecordId)
	}

	// build message
	msg := timelock.NewTimeUnlockMsg(from, timeLockId)
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func queryTimeLocksCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-time-locks",
		Short: "query time locks",
		RunE:  cmdr.queryTimeLocks,
	}

	cmd.Flags().String(flagAccount, "", "account to query")

	return cmd
}

func (c Commander) queryTimeLocks(cmd *cobra.Command, args []string) error {
	cliCtx, _ := client.PrepareCtx(c.Cdc)

	accountStr := viper.GetString(flagAccount)
	account, err := sdk.AccAddressFromBech32(accountStr)
	if err != nil {
		return err
	}

	params := timelock.QueryTimeLocksParams{
		Account: account,
	}

	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return err
	}

	res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", timelock.MsgRoute, timelock.QueryTimeLocks), bz)
	if err != nil {
		return err
	}

	fmt.Println(string(res))
	return nil
}

func queryTimeLockCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-time-lock",
		Short: "query time lock",
		RunE:  cmdr.queryTimeLock,
	}

	cmd.Flags().String(flagAccount, "", "account to query")
	cmd.Flags().String(flagTimeLockId, "", "time lock id")

	return cmd
}

func (c Commander) queryTimeLock(cmd *cobra.Command, args []string) error {
	cliCtx, _ := client.PrepareCtx(c.Cdc)

	accountStr := viper.GetString(flagAccount)
	account, err := sdk.AccAddressFromBech32(accountStr)
	if err != nil {
		return err
	}

	timeLockId := viper.GetInt64(flagTimeLockId)
	if timeLockId < timelock.InitialRecordId {
		return fmt.Errorf("lock time should not less than %d", timelock.InitialRecordId)
	}

	params := timelock.QueryTimeLockParams{
		Account: account,
		Id:      timeLockId,
	}

	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return err
	}

	res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", timelock.MsgRoute, timelock.QueryTimeLock), bz)
	if err != nil {
		return err
	}

	fmt.Println(string(res))
	return nil
}
