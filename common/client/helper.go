package client

import (
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/cosmos/cosmos-sdk/client/context"
	txutils "github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/pkg/errors"
)

func PrepareCtx(cdc *codec.Codec) (context.CLIContext, txbuilder.TxBuilder) {
	txBldr := txbuilder.NewTxBuilderFromCLI().WithCodec(cdc)
	cliCtx := context.NewCLIContext().
		WithCodec(cdc).
		WithAccountDecoder(types.GetAccountDecoder(cdc))
	return cliCtx, txBldr
}

func EnsureSequence(cliCtx context.CLIContext, txBldr *txbuilder.TxBuilder) error {
	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	if txBldr.Sequence == 0 {
		accSeq, err := cliCtx.GetAccountSequence(from)
		if err != nil {
			return err
		}
		*txBldr = txBldr.WithSequence(accSeq)
	}
	return nil
}

func BuildUnsignedTx(builder txbuilder.TxBuilder, acc auth.Account, msgs []sdk.Msg) (*[]byte, error) {
	chainID := builder.ChainID
	if chainID == "" {
		return nil, errors.Errorf("chain ID required but not specified")
	}
	accnum := acc.GetAccountNumber()
	sequence := acc.GetSequence()
	memo := builder.Memo

	signMsg := 	txbuilder.StdSignMsg {
		ChainID:       chainID,
		AccountNumber: accnum,
		Sequence:      sequence,
		Msgs:          msgs,
		Memo:          memo,
	}
	// sign and build
	bz := signMsg.Bytes()
	return &bz, nil
}

func SendOrPrintTx(ctx context.CLIContext, builder txbuilder.TxBuilder, msg sdk.Msg) error {
	if ctx.GenerateOnly {
		return txutils.PrintUnsignedStdTx(builder, ctx, []sdk.Msg{msg}, false)
	}

	return txutils.CompleteAndBroadcastTxCli(builder, ctx, []sdk.Msg{msg})
}
