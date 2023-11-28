package cli

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"

	airdrop "github.com/bnb-chain/node/plugins/recover"
)

const (
	flagAmount      = "amount"
	flagTokenSymbol = "token-symbol"
	flagRecipient   = "recipient"
)

func SignTokenRecoverRequestCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-token-recover-request",
		Short: "get token recover request sign data",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			amount := viper.GetInt64(flagAmount)
			tokenSymbol := viper.GetString(flagTokenSymbol)
			recipient := viper.GetString(flagRecipient)
			msg := airdrop.NewTokenRecoverRequestMsg(tokenSymbol, uint64(amount), strings.ToLower(common.HexToAddress(recipient).Hex()))

			sdkErr := msg.ValidateBasic()
			if sdkErr != nil {
				return fmt.Errorf("%v", sdkErr.Data())
			}
			return SignAndPrint(cliCtx, txBldr, msg)
		},
	}

	cmd.Flags().String(flagTokenSymbol, "", "owner token symbol")
	cmd.Flags().Int64(flagAmount, 0, "amount of token")
	cmd.Flags().String(flagRecipient, "", "bsc recipient address")

	return cmd
}

func SignAndPrint(ctx context.CLIContext, builder authtxb.TxBuilder, msg sdk.Msg) error {
	name, err := ctx.GetFromName()
	if err != nil {
		return err
	}

	passphrase, err := keys.GetPassphrase(name)
	if err != nil {
		return err
	}

	// build and sign the transaction
	stdMsg, err := builder.Build([]sdk.Msg{msg})
	if err != nil {
		return err
	}
	txBytes, err := builder.Sign(name, passphrase, stdMsg)
	if err != nil {
		return err
	}

	var tx auth.StdTx
	if err = builder.Codec.UnmarshalBinaryLengthPrefixed(txBytes, &tx); err == nil {
		json, err := builder.Codec.MarshalJSON(tx)
		if err == nil {
			fmt.Printf("TX JSON: %s\n", json)
		}
	}
	hexBytes := make([]byte, len(txBytes)*2)
	hex.Encode(hexBytes, txBytes)
	txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
	fmt.Printf("Transaction hash: %s, Transaction hex: %s\n", txHash, hexBytes)

	fmt.Println("Sign Message: ", string(stdMsg.Bytes()))
	fmt.Println("Sign Message Hash: ", hex.EncodeToString(crypto.Sha256(stdMsg.Bytes())))
	sig := tx.GetSignatures()[0]
	fmt.Printf("Signature: %s\n", hex.EncodeToString(sig.Signature))
	var originPubKey secp256k1.PubKeySecp256k1
	err = builder.Codec.UnmarshalBinaryBare(sig.PubKey.Bytes(), &originPubKey)
	if err != nil {
		return err
	}
	fmt.Printf("PubKey: %s\n", hex.EncodeToString(originPubKey))
	return nil
}
