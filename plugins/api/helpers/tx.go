package helpers

import (
	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authctx "github.com/cosmos/cosmos-sdk/x/auth/client/context"
)

// BuildUnsignedTx builds a transaction with the given msgs
func BuildUnsignedTx(
	ctx context.CLIContext, acc auth.Account, msgs []sdk.Msg, cdc *wire.Codec,
) (*[]byte, error) {
	txCtx := authctx.NewTxContextFromCLI().WithCodec(cdc)
	chainID := txCtx.ChainID
	if chainID == "" {
		return nil, errors.Errorf("chain ID required but not specified")
	}
	accnum := acc.GetAccountNumber()
	sequence := acc.GetSequence()
	memo := txCtx.Memo

	// TODO: add the fee
	fee := sdk.Coin{}
	if txCtx.Fee != "" {
		parsedFee, err := sdk.ParseCoin(txCtx.Fee)
		if err != nil {
			return nil, err
		}
		fee = parsedFee
	}

	signMsg := auth.StdSignMsg{
		ChainID:       chainID,
		AccountNumber: accnum,
		Sequence:      sequence,
		Msgs:          msgs,
		Memo:          memo,
		Fee:           auth.NewStdFee(txCtx.Gas, fee), // TODO run simulate to estimate gas?
	}

	// sign and build
	bz := signMsg.Bytes()

	return &bz, nil
}
