package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bnb-chain/node/common"
	"github.com/bnb-chain/node/common/client"
	"github.com/bnb-chain/node/plugins/tokens/swap"
)

const (
	flagCreatorAddr         = "creator-addr"
	flagRecipientAddr       = "recipient-addr"
	flagExpectedIncome      = "expected-income"
	flagRecipientOtherChain = "recipient-other-chain"
	flagSenderOtherChain    = "sender-other-chain"
	flagRandomNumberHash    = "random-number-hash"
	flagRandomNumber        = "random-number"
	flagSwapID              = "swap-id"
	flagTimestamp           = "timestamp"
	flagHeightSpan          = "height-span"
	flagCrossChain          = "cross-chain"
	flagLimit               = "limit"
	flagOffset              = "offset"
)

func initiateHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "HTLT",
		Short: "Create a hash timer lock transfer",
		RunE:  cmdr.initiateHTLT,
	}

	cmd.Flags().String(flagRecipientAddr, "", "The recipient address of BEP2 token, bech32 encoding")
	cmd.Flags().String(flagAmount, "", "The swapped out amount BEP2 tokens, example: \"100:BNB\" or \"100:BNB,10000:BTCB-1DE\"")
	cmd.Flags().String(flagExpectedIncome, "", "Expected income from swap counter party, example: \"100:BNB\" or \"100:BNB,10000:BTCB-1DE\"")
	cmd.Flags().String(flagRecipientOtherChain, "", "The recipient address on other chain, like Ethereum, leave it empty for single chain swap")
	cmd.Flags().String(flagSenderOtherChain, "", "The sender address on other chain, like Ethereum, leave it empty for single chain swap")
	cmd.Flags().String(flagRandomNumberHash, "", "RandomNumberHash of random number and timestamp, based on SHA256, 32 bytes, hex encoding. If left out, a random value will be generated")
	cmd.Flags().Int64(flagTimestamp, 0, "The time of sending transaction, counted by second. In the response to a swap request from other chains, it should be identical to the one in the swap request. If left out, current timestamp will be used")
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
	amount, err := sdk.ParseCoins(viper.GetString(flagAmount))
	if err != nil {
		return err
	}

	expectedIncome := viper.GetString(flagExpectedIncome)
	recipientOtherChain := viper.GetString(flagRecipientOtherChain)
	senderOtherChain := viper.GetString(flagSenderOtherChain)

	var timestamp int64
	var randomNumberHash []byte
	timestamp = viper.GetInt64(flagTimestamp)
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	randomNumberHashStr := viper.GetString(flagRandomNumberHash)
	if len(randomNumberHashStr) == 0 {
		randomNumber := make([]byte, swap.RandomNumberLength)
		length, err := rand.Read(randomNumber)
		if err != nil || length != swap.RandomNumberLength {
			return fmt.Errorf("failed to generate random number")
		}
		randomNumberHash = swap.CalculateRandomHash(randomNumber, timestamp)
		fmt.Println(fmt.Sprintf("Random number: %s", hex.EncodeToString(randomNumber)))
	} else {
		randomNumberHash, err = hex.DecodeString(randomNumberHashStr)
		if err != nil {
			return err
		}
	}
	fmt.Println(fmt.Sprintf("Timestamp: %d\nRandom number hash: %s", timestamp, hex.EncodeToString(randomNumberHash)))
	heightSpan := viper.GetInt64(flagHeightSpan)
	crossChain := viper.GetBool(flagCrossChain)
	// build message
	msg := swap.NewHTLTMsg(from, to, recipientOtherChain, senderOtherChain, randomNumberHash, timestamp, amount, expectedIncome, heightSpan, crossChain)

	sdkErr := msg.ValidateBasic()
	if sdkErr != nil {
		return fmt.Errorf("%v", sdkErr.Data())
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func depositHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit",
		Short: "deposit a hash timer lock transfer",
		RunE:  cmdr.depositHTLT,
	}

	cmd.Flags().String(flagAmount, "", "The swapped out amount BEP2 tokens, example: \"100:BNB\" or \"100:BNB,10000:BTCB-1DE\"")
	cmd.Flags().String(flagSwapID, "", "ID of previously created swap, hex encoding")

	return cmd
}

func (c Commander) depositHTLT(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	amount, err := sdk.ParseCoins(viper.GetString(flagAmount))
	if err != nil {
		return err
	}

	swapID, err := hex.DecodeString(viper.GetString(flagSwapID))
	if err != nil {
		return err
	}

	// build message
	msg := swap.NewDepositHTLTMsg(from, amount, swapID)

	sdkErr := msg.ValidateBasic()
	if sdkErr != nil {
		return fmt.Errorf("%v", sdkErr.Data())
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func claimHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim",
		Short: "claim a hash timer lock transfer",
		RunE:  cmdr.claimHTLT,
	}

	cmd.Flags().String(flagSwapID, "", "ID of previously created swap, hex encoding")
	cmd.Flags().String(flagRandomNumber, "", "The random number to unlock the locked hash, 32 bytes, hex encoding")

	return cmd
}

func (c Commander) claimHTLT(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	swapID, err := hex.DecodeString(viper.GetString(flagSwapID))
	if err != nil {
		return err
	}

	randomNumber, err := hex.DecodeString(viper.GetString(flagRandomNumber))
	if err != nil {
		return err
	}

	// build message
	msg := swap.NewClaimHTLTMsg(from, swapID, randomNumber)

	sdkErr := msg.ValidateBasic()
	if sdkErr != nil {
		return fmt.Errorf("%v", sdkErr.Data())
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func refundHTLTCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refund",
		Short: "refund a hash timer lock transfer",
		RunE:  cmdr.refundHTLT,
	}

	cmd.Flags().String(flagSwapID, "", "ID of previously created swap, hex encoding")

	return cmd
}

func (c Commander) refundHTLT(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	swapID, err := hex.DecodeString(viper.GetString(flagSwapID))
	if err != nil {
		return err
	}

	// build message
	msg := swap.NewRefundHTLTMsg(from, swapID)

	sdkErr := msg.ValidateBasic()
	if sdkErr != nil {
		return fmt.Errorf("%v", sdkErr.Data())
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func querySwapCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-swap",
		Short: "Query an atomic swap by swapID",
		RunE:  cmdr.querySwap,
	}

	cmd.Flags().String(flagSwapID, "", "ID of previously created swap, hex encoding")

	return cmd
}

func (c Commander) querySwap(cmd *cobra.Command, args []string) error {

	cliCtx, _ := client.PrepareCtx(c.Cdc)

	swapID, err := hex.DecodeString(viper.GetString(flagSwapID))
	if err != nil {
		return err
	}
	if len(swapID) != swap.SwapIDLength {
		return fmt.Errorf("expected swapID length is %d, actually it is %d", swap.SwapIDLength, len(swapID))
	}

	hashKey := swap.BuildHashKey(swapID)

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
		Use:   "query-swapIDs-by-creator",
		Short: "Query swapID list by the creator address",
		RunE:  cmdr.querySwapsByCreator,
	}

	cmd.Flags().String(flagCreatorAddr, "", "Swap creator address, bech32 encoding")
	cmd.Flags().Int64(flagLimit, 100, "The maximum quantity of swapIDs you want to get")
	cmd.Flags().Int64(flagOffset, 0, "The number of swapIDs you want to skip")

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
	if limit <= 0 || limit > 100 {
		return fmt.Errorf("limit should be (1, 100]")
	}
	if offset < 0 {
		return fmt.Errorf("offset must be positive")
	}

	params := swap.QuerySwapByCreatorParams{
		Creator: creator,
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
		Use:   "query-swapIDs-by-recipient",
		Short: "Query swapID list by the recipient address",
		RunE:  cmdr.querySwapsByRecipient,
	}

	cmd.Flags().String(flagRecipientAddr, "", "Swap recipient address, bech32 encoding")
	cmd.Flags().Int64(flagLimit, 100, "The maximum quantity of swapIDs you want to get")
	cmd.Flags().Int64(flagOffset, 0, "The number of swapIDs you want to skip")

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
	if limit <= 0 || limit > 100 {
		return fmt.Errorf("limit should be (1, 100]")
	}
	if offset < 0 {
		return fmt.Errorf("offset must be positive")
	}

	params := swap.QuerySwapByRecipientParams{
		Recipient: recipient,
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
