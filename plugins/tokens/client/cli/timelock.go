package commands

import (
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/plugins/tokens/timelock"

	"github.com/binance-chain/node/common/client"
)

const (
	flagLockTime         = "lock-time"
	flagDescription      = "description"
	flagTimeLockId       = "time-lock-id"
	flagAddress          = "address"
	flagIncreaseAmountTo = "increase-amount-to"
	flagExtendedLockTime = "extended-lock-time"
	flagBroadcast        = "broadcast"
)

func timeLockCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time-lock",
		Short: "time lock tokens",
		Long: strings.TrimSpace(`
Time lock is to lock an amount of tokens to a given time before when these tokens will not be able to claim back.

$ CLI token time-lock --amount 100:BNB --from alice --description "time lock for some reason" --lock-time 1559805558

this command will output the tx to broadcast, but will not broadcast to any node. you can double check the tx msg before send it to blockchain.

if you want to broadcast the tx to blockchain, you need to specify --broadcast manually.

$ CLI token time-lock --amount 100:BNB --from alice --description "time lock for some reason" --lock-time 1559805558 --broadcast
`),
		RunE: cmdr.timeLock,
	}

	cmd.Flags().String(flagAmount, "", "amount of tokens to lock")
	cmd.Flags().Int64(flagLockTime, 0, "timestamp of lock time(second)")
	cmd.Flags().String(flagDescription, "", "description of time lock")
	cmd.Flags().Bool(flagBroadcast, false, "broadcast tx")

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
	broadcast := viper.GetBool(flagBroadcast)
	if !broadcast {
		cliCtx.GenerateOnly = true
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func timeRelockCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time-relock",
		Short: "time relock tokens",
		Long: strings.TrimSpace(`
Time relock is used to extend the lock time, increase amount of locked tokens or modify the description of a given time lock record.

$ CLI token time-lock --from alice --description "time lock for some reason" --extended-lock-time 1559805558 --increase-amount-to 1000:BNB --time-lock-id 1

this command will output the tx to broadcast, but will not broadcast to any node. you can double check the tx msg before send it to blockchain.

you may just want to change description, extend lock time or increase locked amount, so you just need to specify one flag one time(but at least one).

for example:

$ CLI token time-lock --from alice --extended-lock-time 1559805558 --time-lock-id 1

this command will just extend the lock time of this time lock record.

if you want to broadcast the tx to blockchain, you need to specify --broadcast manually.

$ CLI token time-lock --from alice --description "time lock for some reason" --extended-lock-time 1559805558 --increase-amount-to 1000:BNB --time-lock-id 1 --broadcast
`),
		RunE: cmdr.timeRelock,
	}

	cmd.Flags().String(flagIncreaseAmountTo, "", "amount of tokens to lock")
	cmd.Flags().Int64(flagExtendedLockTime, 0, "timestamp of lock time(second)")
	cmd.Flags().String(flagDescription, "", "description of time lock")
	cmd.Flags().Int64(flagTimeLockId, 0, "time lock id")
	cmd.Flags().Bool(flagBroadcast, false, "broadcast tx")

	return cmd
}

func (c Commander) timeRelock(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)
	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	timeLockId := viper.GetInt64(flagTimeLockId)
	if timeLockId < timelock.InitialRecordId {
		return fmt.Errorf("time lock id should not less than %d", timelock.InitialRecordId)
	}

	description := viper.GetString(flagDescription)

	if len(description) > timelock.MaxDescriptionLength {
		return fmt.Errorf("length of description should be less than %d", timelock.MaxDescriptionLength)
	}

	amount, err := sdk.ParseCoins(viper.GetString(flagIncreaseAmountTo))
	if err != nil {
		return err
	}

	lockTime := viper.GetInt64(flagExtendedLockTime)
	if lockTime < 0 {
		return fmt.Errorf("lock time should be positive")
	}

	if lockTime != 0 && time.Unix(lockTime, 0).Before(time.Now()) {
		return fmt.Errorf("lock time(%s) should be after now", time.Unix(lockTime, 0).UTC().String())
	}

	if len(description) == 0 &&
		amount.IsZero() &&
		lockTime == 0 {
		return fmt.Errorf("no thing specified to update on original time lock")
	}

	// build message
	msg := timelock.NewTimeRelockMsg(from, timeLockId, description, amount, lockTime)
	broadcast := viper.GetBool(flagBroadcast)
	if !broadcast {
		cliCtx.GenerateOnly = true
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func timeUnlockCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time-unlock",
		Short: "time unlock tokens",
		RunE:  cmdr.timeUnlock,
	}

	cmd.Flags().Int64(flagTimeLockId, 0, "time lock id")

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
		return fmt.Errorf("time lock id should not less than %d", timelock.InitialRecordId)
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

	cmd.Flags().String(flagAddress, "", "address to query")

	return cmd
}

func (c Commander) queryTimeLocks(cmd *cobra.Command, args []string) error {
	cliCtx, _ := client.PrepareCtx(c.Cdc)

	addressStr := viper.GetString(flagAddress)
	address, err := sdk.AccAddressFromBech32(addressStr)
	if err != nil {
		return err
	}

	params := timelock.QueryTimeLocksParams{
		Account: address,
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

	cmd.Flags().String(flagAddress, "", "address to query")
	cmd.Flags().Int64(flagTimeLockId, 0, "time lock id")

	return cmd
}

func (c Commander) queryTimeLock(cmd *cobra.Command, args []string) error {
	cliCtx, _ := client.PrepareCtx(c.Cdc)

	addressStr := viper.GetString(flagAddress)
	address, err := sdk.AccAddressFromBech32(addressStr)
	if err != nil {
		return err
	}

	timeLockId := viper.GetInt64(flagTimeLockId)
	if timeLockId < timelock.InitialRecordId {
		return fmt.Errorf("lock time should not less than %d", timelock.InitialRecordId)
	}

	params := timelock.QueryTimeLockParams{
		Account: address,
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
