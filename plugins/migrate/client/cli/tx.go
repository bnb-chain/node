package cli

import (
	"encoding/hex"
	"fmt"

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

	"github.com/bnb-chain/node/plugins/migrate"
)

const (
	flagBSCOperatorAddress = "bsc-operator-address"
)

func SignValidatorOwnerShipCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-validator-ownership",
		Short: "get validator ownership sign data",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			bscOperator := viper.GetString(flagBSCOperatorAddress)

			return SignAndPrint(cliCtx, txBldr, common.HexToAddress(bscOperator))
		},
	}

	cmd.Flags().String(flagBSCOperatorAddress, "", "bsc operator address")

	return cmd
}

func SignAndPrint(ctx context.CLIContext, builder authtxb.TxBuilder, bscOperator common.Address) error {
	name, err := ctx.GetFromName()
	if err != nil {
		return err
	}

	passphrase, err := keys.GetPassphrase(name)
	if err != nil {
		return err
	}

	msg := migrate.NewValidatorOwnerShipMsg(bscOperator)

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
	err = builder.Codec.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
	if err != nil {
		return err
	}
	json, err := builder.Codec.MarshalJSON(tx)
	if err != nil {
		return err
	}

	fmt.Printf("TX JSON: %s\n", json)
	fmt.Println("Sign Message: ", string(stdMsg.Bytes()))
	fmt.Println("Sign Message Hash: ", "0x"+hex.EncodeToString(crypto.Sha256(stdMsg.Bytes())))
	sig := tx.GetSignatures()[0]
	fmt.Printf("Signature: %s\n", "0x"+hex.EncodeToString(sig.Signature))
	var originPubKey secp256k1.PubKeySecp256k1
	err = builder.Codec.UnmarshalBinaryBare(sig.PubKey.Bytes(), &originPubKey)
	if err != nil {
		return err
	}
	fmt.Printf("PubKey: %s\n", "0x"+hex.EncodeToString(originPubKey))
	return nil
}
