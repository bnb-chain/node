package commands

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/plugins/tokens/swap"
)

const (
	flagAuto             = "auto"
	flagFromAddr         = "from-addr"
	flagToAddr           = "to-addr"
	flagOutAmount        = "out-amount"
	flagInAmount         = "in-amount"
	flagToOnOtherChain   = "to-on-other-chain"
	flagRandomNumberHash = "random-number-hash"
	flagRandomNumber     = "random-number"
	flagTimestamp        = "timestamp"
	flagTimespan         = "timespan"
	flagLimit            = "limit"
	flagOffset           = "offset"
	flagStatus           = "swap-status"
)

func initiateSwapCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initiate-swap",
		Short: "initiate an atomic swap",
		RunE:  cmdr.initiateSwap,
	}

	cmd.Flags().Bool(flagAuto, false, "Automatically generate random number hash and timestamp, if true, --random-number-hash and --timestamp can be left out")
	cmd.Flags().String(flagToAddr, "", "The receiver address of BEP2 token, bech32 encoding")
	cmd.Flags().String(flagOutAmount, "", "The swapped out amount BEP2 token, example: 100:BNB")
	cmd.Flags().Int64(flagInAmount, 0, "Expected gained token on the other chain, 8 decimals")
	cmd.Flags().String(flagToOnOtherChain, "", "The receiver address on other chain, like Ethereum, hex encoding and prefix with 0x")
	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")
	cmd.Flags().Int64(flagTimestamp, 0, "The time of sending transaction, counted by second. In the response to a swap request from other chains, it should be identical to the one in the swap request")
	cmd.Flags().Int64(flagTimespan, 0, "The number of blocks to wait before the asset may be returned to swap creator if not claimed via random number")

	return cmd
}

func (c Commander) initiateSwap(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	to, err := sdk.AccAddressFromBech32(viper.GetString(flagToAddr))
	if err != nil {
		return err
	}
	outAmount, err := sdk.ParseCoin(viper.GetString(flagOutAmount))
	if err != nil {
		return err
	}

	inAmount := viper.GetInt64(flagInAmount)
	toOnOtherChainStr := viper.GetString(flagToOnOtherChain)
	if !strings.HasPrefix(toOnOtherChainStr, "0x") {
		return fmt.Errorf("must specify hex encoding string and prefix with 0x for flag --to-on-other-chain")
	}
	toOnOtherChain, err := hex.DecodeString(toOnOtherChainStr[2:])
	if err != nil {
		return err
	}

	var randomNumberHash []byte
	var timestamp int64
	if !viper.GetBool(flagAuto) {
		randomNumberHashStr := viper.GetString(flagRandomNumberHash)
		if !strings.HasPrefix(randomNumberHashStr, "0x") {
			return fmt.Errorf("must specify hex encoding string and prefix with 0x for flag --random-number-hash")
		}
		randomNumberHash, err = hex.DecodeString(randomNumberHashStr[2:])
		if err != nil {
			return err
		}
		timestamp = viper.GetInt64(flagTimestamp)
	} else {
		randomeNumber := make([]byte, swap.RandomNumberLength)
		rand.Read(randomeNumber)
		timestamp = time.Now().Unix()
		randomNumberHash = swap.CalculteRandomHash(randomeNumber, timestamp)

		fmt.Println(fmt.Sprintf("Random number: 0x%s \nTimestamp: %d \nRandom number hash: 0x%s", hex.EncodeToString(randomeNumber), timestamp, hex.EncodeToString(randomNumberHash)))
	}
	timespan := viper.GetInt64(flagTimespan)
	// build message
	msg := swap.NewHashTimerLockTransferMsg(from, to, toOnOtherChain, randomNumberHash, timestamp, outAmount, inAmount, timespan)

	err = msg.ValidateBasic()
	if err != nil {
		return err
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func claimSwapCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-swap",
		Short: "claim an atomic swap with random number",
		RunE:  cmdr.claimSwap,
	}

	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")
	cmd.Flags().String(flagRandomNumber, "", "The secret random number, 32 bytes, hex encoding and prefix with 0x")

	return cmd
}

func (c Commander) claimSwap(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	randomNumberHashStr := viper.GetString(flagRandomNumberHash)
	if !strings.HasPrefix(randomNumberHashStr, "0x") {
		return fmt.Errorf("must specify hex encoding string and prefix with 0x for flag --random-number-hash")
	}
	randomNumberHash, err := hex.DecodeString(randomNumberHashStr[2:])
	if err != nil {
		return err
	}

	randomNumberStr := viper.GetString(flagRandomNumber)
	if !strings.HasPrefix(randomNumberStr, "0x") {
		return fmt.Errorf("must specify hex encoding string and prefix with 0x for flag --random-number")
	}
	randomNumber, err := hex.DecodeString(randomNumberStr[2:])
	if err != nil {
		return err
	}

	// build message
	msg := swap.NewClaimHashTimerLockMsg(from, randomNumberHash, randomNumber)

	err = msg.ValidateBasic()
	if err != nil {
		return err
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func refundSwapCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refund-swap",
		Short: "refund the asset locked by an expired atomic swap",
		RunE:  cmdr.refundSwap,
	}

	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")

	return cmd
}

func (c Commander) refundSwap(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	randomNumberHashStr := viper.GetString(flagRandomNumberHash)
	if !strings.HasPrefix(randomNumberHashStr, "0x") {
		return fmt.Errorf("must specify hex encoding string and prefix with 0x for flag --random-number-hash")
	}
	randomNumberHash, err := hex.DecodeString(randomNumberHashStr[2:])
	if err != nil {
		return err
	}

	// build message
	msg := swap.NewRefundLockedAssetMsg(from, randomNumberHash)

	err = msg.ValidateBasic()
	if err != nil {
		return err
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func querySwapCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-swap",
		Short: "Query an atomic swap by random number hash",
		RunE:  cmdr.querySwap,
	}

	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")

	return cmd
}

func (c Commander) querySwap(cmd *cobra.Command, args []string) error {

	cliCtx, _ := client.PrepareCtx(c.Cdc)

	randomNumberHashStr := viper.GetString(flagRandomNumberHash)
	if !strings.HasPrefix(randomNumberHashStr, "0x") {
		return fmt.Errorf("must specify hex encoding string and prefix with 0x for flag --random-number-hash")
	}
	randomNumberHash, err := hex.DecodeString(randomNumberHashStr[2:])
	if err != nil {
		return err
	}

	hashKey := swap.GetSwapHashKey(randomNumberHash)

	res, err := cliCtx.QueryStore(hashKey, common.AtomicSwapStoreName)
	if err != nil {
		return err
	}

	if res == nil {
		return fmt.Errorf("no matched swap record")
	}

	atomicSwap := swap.DecodeAtomicSwap(c.Cdc, res)
	var output []byte
	if cliCtx.Indent {
		output, err = c.Cdc.MarshalJSONIndent(atomicSwap, "", "  ")
	} else {
		output, err = c.Cdc.MarshalJSON(atomicSwap)
	}
	fmt.Println(string(output))

	return nil
}

func querySwapsFromCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-swap-from",
		Short: "Query swaps from the specified address",
		RunE:  cmdr.querySwapsFrom,
	}

	cmd.Flags().String(flagFromAddr, "", "The from address of swap, bech32 encoding")
	cmd.Flags().Int64(flagLimit, 100, "query result limitation")
	cmd.Flags().Int64(flagOffset, 0, "skipped quantity")
	cmd.Flags().String(flagStatus, "NULL", "Swap status, NULL|Open|Completed|Expired")

	return cmd
}

func (c Commander) querySwapsFrom(cmd *cobra.Command, args []string) error {

	cliCtx, _ := client.PrepareCtx(c.Cdc)

	fromAddr, err := sdk.AccAddressFromBech32(viper.GetString(flagFromAddr))
	if err != nil {
		return err
	}
	limit := viper.GetInt64(flagLimit)
	offset := viper.GetInt64(flagOffset)
	swapStatus := swap.NewSwapStatusFromString(viper.GetString(flagStatus))

	params := swap.QuerySwapFromParams{
		From:   fromAddr,
		Status: swapStatus,
		Limit:  limit,
		Offset: offset,
	}

	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return err
	}

	res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapFrom), bz)
	if err != nil {
		return err
	}

	fmt.Println(string(res))
	return nil
}

func querySwapsToCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-swap-to",
		Short: "Query swaps to the specified address",
		RunE:  cmdr.querySwapsTo,
	}

	cmd.Flags().String(flagToAddr, "", "The receiver address of swap, bech32 encoding")
	cmd.Flags().Int64(flagLimit, 100, "query result limitation")
	cmd.Flags().Int64(flagOffset, 0, "skipped quantity")
	cmd.Flags().String(flagStatus, "NULL", "Swap status, NULL|Open|Completed|Expired")

	return cmd
}

func (c Commander) querySwapsTo(cmd *cobra.Command, args []string) error {

	cliCtx, _ := client.PrepareCtx(c.Cdc)

	toAddr, err := sdk.AccAddressFromBech32(viper.GetString(flagToAddr))
	if err != nil {
		return err
	}
	limit := viper.GetInt64(flagLimit)
	offset := viper.GetInt64(flagOffset)
	swapStatus := swap.NewSwapStatusFromString(viper.GetString(flagStatus))

	params := swap.QuerySwapToParams{
		To:     toAddr,
		Status: swapStatus,
		Limit:  limit,
		Offset: offset,
	}

	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return err
	}

	res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapTo), bz)
	if err != nil {
		return err
	}

	fmt.Println(string(res))
	return nil
}
