package helpers

import (
	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// BuildUnsignedTx builds a transaction with the given msgs
func BuildUnsignedTx(
	ctx context.CoreContext, acc auth.Account, msgs []sdk.Msg, cdc *wire.Codec,
) (*[]byte, error) {
	chainID := ctx.ChainID
	if chainID == "" {
		return nil, errors.Errorf("chain ID required but not specified")
	}
	accnum := acc.GetAccountNumber()
	sequence := acc.GetSequence()
	memo := ctx.Memo

	// TODO: add the fee
	fee := sdk.Coin{}
	if ctx.Fee != "" {
		parsedFee, err := sdk.ParseCoin(ctx.Fee)
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
		Fee:           auth.NewStdFee(ctx.Gas, fee), // TODO run simulate to estimate gas?
	}

	// sign and build
	bz := signMsg.Bytes()

	return &bz, nil
}
