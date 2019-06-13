package validation

import (
	"bytes"

	cmntypes "github.com/binance-chain/node/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

const (
	TransferMemoCheckerFlag = 0x0000000000000001
)

func CustomizedValidation(ctx sdk.Context, am auth.AccountKeeper, tx auth.StdTx) sdk.Error {
	for _, addr := range tx.Msgs[0].GetInvolvedAddresses() {
		acc := am.GetAccount(ctx, addr)
		if acc == nil {
			return nil
		}
		account, ok := acc.(cmntypes.NamedAccount)
		if !ok {
			return nil
		}
		flags := account.GetFlags()

		// transfer memo checker
		if flags&TransferMemoCheckerFlag > 0 {
			err := transferMemoChecker(addr, tx)
			if err != nil {
				return err
			}
		}

		// Add more validation here

	}
	return nil
}

func transferMemoChecker(addr sdk.AccAddress, tx auth.StdTx) sdk.Error {
	//transaction type check
	if tx.Msgs[0].Type() != "send" {
		return nil
	}
	//check if addr is receiver
	sendMsg := tx.Msgs[0].(bank.MsgSend)
	isReceiver := false
	for _, out := range sendMsg.Outputs {
		if bytes.Equal(out.Address, addr) {
			isReceiver = true
		}
	}
	if !isReceiver {
		return nil
	}
	if len(tx.Memo) == 0 {
		return sdk.ErrInvalidTxMemo("receiver requires non-empty memo in transfer transaction")
	}
	for index := 0; index < len(tx.Memo); index++ {
		if tx.Memo[index] > '9' || tx.Memo[index] < '0' {
			return sdk.ErrInvalidTxMemo("receiver requires that memo only contains digital character")
		}
	}
	return nil
}
