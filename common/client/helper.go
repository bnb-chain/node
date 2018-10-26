package client

import (
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/cosmos/cosmos-sdk/client/context"
	txutils "github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
)

func PrepareCtx(cdc *codec.Codec) (context.CLIContext, txbuilder.TxBuilder) {
	txBldr := txbuilder.NewTxBuilderFromCLI().WithCodec(cdc)
	cliCtx := context.NewCLIContext().
		WithCodec(cdc).
		WithAccountDecoder(types.GetAccountDecoder(cdc))
	return cliCtx, txBldr
}

func SendOrPrintTx(ctx context.CLIContext, builder txbuilder.TxBuilder, msg sdk.Msg) error {
	if ctx.GenerateOnly {
		return txutils.PrintUnsignedStdTx(builder, ctx, []sdk.Msg{msg}, false)
	}

	return txutils.CompleteAndBroadcastTxCli(builder, ctx, []sdk.Msg{msg})
}
