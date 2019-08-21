package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/plugins/tokens/swap"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	flagAuto                = "auto"
	flagCreatorAddr         = "creator-addr"
	flagRecipientAddr       = "recipient-addr"
	flagOutAmount           = "out-amount"
	flagExpectedIncome      = "expected-income"
	flagRecipientOtherChain = "recipient-other-chain"
	flagRandomNumberHash    = "random-number-hash"
	flagRandomNumber        = "random-number"
	flagTimestamp           = "timestamp"
	flagHeightSpan          = "height-span"
	flagCrossChain          = "cross-chain"
	flagLimit               = "limit"
	flagOffset              = "offset"
	flagStatus              = "swap-status"
)

func initiateHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "HTLT",
		Short: "Create a hash timer lock transfer",
		RunE:  cmdr.initiateHTLT,
	}

	cmd.Flags().Bool(flagAuto, false, "Automatically generate random number hash and timestamp, if true, --random-number-hash and --timestamp can be left out")
	cmd.Flags().String(flagRecipientAddr, "", "The recipient address of BEP2 token, bech32 encoding")
	cmd.Flags().String(flagOutAmount, "", "The swapped out amount BEP2 token, example: 100:BNB")
	cmd.Flags().String(flagExpectedIncome, "", "Expected income on the other chain")
	cmd.Flags().String(flagRecipientOtherChain, "", "The recipient address on other chain, like Ethereum, hex encoding and prefix with 0x, leave it empty for single chain swap")
	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")
	cmd.Flags().Int64(flagTimestamp, 0, "The time of sending transaction, counted by second. In the response to a swap request from other chains, it should be identical to the one in the swap request")
	cmd.Flags().Int64(flagHeightSpan, 0, "The number of blocks to wait before the asset may be returned to swap creator if not claimed via random number")
	cmd.Flags().Bool(flagCrossChain, false, "Create cross chain hash timer lock transfer")

	return cmd
}

func (c Commander) initiateHTLT(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	to, err := sdk.AccAddressFromBech32(viper.GetString(flagRecipientAddr))
	if err != nil {
		return err
	}
	outAmount, err := sdk.ParseCoin(viper.GetString(flagOutAmount))
	if err != nil {
		return err
	}

	expectedIncome := viper.GetString(flagExpectedIncome)
	recipientOtherChainStr := viper.GetString(flagRecipientOtherChain)

	var recipientOtherChain swap.HexData
	if len(recipientOtherChainStr) !=0 {
		if !strings.HasPrefix(recipientOtherChainStr, "0x") {
			return fmt.Errorf("must prefix with 0x for flag --recipient-other-chain")
		}
		recipientOtherChain, err = hex.DecodeString(recipientOtherChainStr[2:])
		if err != nil {
			return err
		}
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
		randomNumber := make([]byte, swap.RandomNumberLength)
		length, err := rand.Read(randomNumber)
		if err != nil || length != swap.RandomNumberLength {
			return fmt.Errorf("failed to generate random number")
		}
		timestamp = time.Now().Unix()
		randomNumberHash = swap.CalculateRandomHash(randomNumber, timestamp)

		fmt.Println(fmt.Sprintf("Random number: 0x%s \nTimestamp: %d \nRandom number hash: 0x%s", hex.EncodeToString(randomNumber), timestamp, hex.EncodeToString(randomNumberHash)))
	}
	heightSpan := viper.GetInt64(flagHeightSpan)
	crossChain := viper.GetBool(flagCrossChain)
	// build message
	msg := swap.NewHashTimerLockedTransferMsg(from, to, recipientOtherChain, randomNumberHash, timestamp, outAmount, expectedIncome, heightSpan, crossChain)

	err = msg.ValidateBasic()
	if err != nil {
		return err
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func depositHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit-HTLT",
		Short: "deposit a hash timer lock transfer",
		RunE:  cmdr.depositHTLT,
	}

	cmd.Flags().String(flagOutAmount, "", "The swapped out amount BEP2 token, example: 100:BNB")
	cmd.Flags().String(flagRecipientAddr, "", "The recipient address of BEP2 token, bech32 encoding")
	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")

	return cmd
}

func (c Commander) depositHTLT(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	recipient, err := sdk.AccAddressFromBech32(viper.GetString(flagRecipientAddr))
	if err != nil {
		return err
	}
	outAmount, err := sdk.ParseCoin(viper.GetString(flagOutAmount))
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
	msg := swap.NewDepositHashTimerLockedTransferMsg(from, recipient, outAmount, randomNumberHash)

	err = msg.ValidateBasic()
	if err != nil {
		return err
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func claimHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-HTLT",
		Short: "claim a hash timer lock transfer",
		RunE:  cmdr.claimHTLT,
	}

	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")
	cmd.Flags().String(flagRandomNumber, "", "The secret random number, 32 bytes, hex encoding and prefix with 0x")

	return cmd
}

func (c Commander) claimHTLT(cmd *cobra.Command, args []string) error {
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
	msg := swap.NewClaimHashTimerLockedTransferMsg(from, randomNumberHash, randomNumber)

	err = msg.ValidateBasic()
	if err != nil {
		return err
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func refundHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refund-HTLT",
		Short: "refund a hash timer lock transfer",
		RunE:  cmdr.refundHTLT,
	}

	cmd.Flags().String(flagRandomNumberHash, "", "Hash of random number and timestamp, based on SHA256, 32 bytes, hex encoding and prefix with 0x")

	return cmd
}

func (c Commander) refundHTLT(cmd *cobra.Command, args []string) error {
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
	msg := swap.NewRefundRefundHashTimerLockedTransferMsg(from, randomNumberHash)

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

	hashKey := swap.BuildHashKey(randomNumberHash)

	res, err := cliCtx.QueryStore(hashKey, common.AtomicSwapStoreName)
	if err != nil {
		return err
	}

	if res == nil {
		return fmt.Errorf("no matched swap record")
	}

	var atomicSwap swap.AtomicSwap
	c.Cdc.MustUnmarshalBinaryBare(res, &atomicSwap)
	var output []byte
	if cliCtx.Indent {
		output, err = c.Cdc.MarshalJSONIndent(atomicSwap, "", "  ")
	} else {
		output, err = c.Cdc.MarshalJSON(atomicSwap)
	}
	fmt.Println(string(output))

	return nil
}

func querySwapsByCreatorCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-swap-by-creator",
		Short: "Query swaps by the creator address",
		RunE:  cmdr.querySwapsByCreator,
	}

	cmd.Flags().String(flagCreatorAddr, "", "Swap creator address, bech32 encoding")
	cmd.Flags().Int64(flagLimit, 100, "query result limitation")
	cmd.Flags().Int64(flagOffset, 0, "skipped quantity")
	cmd.Flags().String(flagStatus, "NULL", "Swap status, NULL|Open|Completed|Expired")

	return cmd
}

func (c Commander) querySwapsByCreator(cmd *cobra.Command, args []string) error {

	cliCtx, _ := client.PrepareCtx(c.Cdc)

	creator, err := sdk.AccAddressFromBech32(viper.GetString(flagCreatorAddr))
	if err != nil {
		return err
	}
	limit := viper.GetInt64(flagLimit)
	offset := viper.GetInt64(flagOffset)
	swapStatus := swap.NewSwapStatusFromString(viper.GetString(flagStatus))

	params := swap.QuerySwapByCreatorParams{
		Creator: creator,
		Status:  swapStatus,
		Limit:   limit,
		Offset:  offset,
	}

	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return err
	}

	res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapCreator), bz)
	if err != nil {
		return err
	}

	fmt.Println(string(res))
	return nil
}

func querySwapsByRecipientCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-swap-by-recipient",
		Short: "Query swaps by the recipient address",
		RunE:  cmdr.querySwapsByRecipient,
	}

	cmd.Flags().String(flagRecipientAddr, "", "Swap recipient address, bech32 encoding")
	cmd.Flags().Int64(flagLimit, 100, "query result limitation")
	cmd.Flags().Int64(flagOffset, 0, "skipped quantity")
	cmd.Flags().String(flagStatus, "NULL", "Swap status, NULL|Open|Completed|Expired")

	return cmd
}

func (c Commander) querySwapsByRecipient(cmd *cobra.Command, args []string) error {

	cliCtx, _ := client.PrepareCtx(c.Cdc)

	recipient, err := sdk.AccAddressFromBech32(viper.GetString(flagRecipientAddr))
	if err != nil {
		return err
	}
	limit := viper.GetInt64(flagLimit)
	offset := viper.GetInt64(flagOffset)
	swapStatus := swap.NewSwapStatusFromString(viper.GetString(flagStatus))

	params := swap.QuerySwapByRecipientParams{
		Recipient: recipient,
		Status:    swapStatus,
		Limit:     limit,
		Offset:    offset,
	}

	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return err
	}

	res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", swap.AtomicSwapRoute, swap.QuerySwapRecipient), bz)
	if err != nil {
		return err
	}

	fmt.Println(string(res))
	return nil
}
