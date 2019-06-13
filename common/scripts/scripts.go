package scripts

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	cmntypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
)

const (
	TransferMemoCheckerFlag = 0x0000000000000001  // uint64 BEP12
)

func AddTransferMemoCheckScript(am auth.AccountKeeper, txDecoder sdk.TxDecoder) {
	msgType := bank.MsgSend{}.Type()
	auth.RegisterScripts(msgType, generateTransferMemoCheckScript(am, txDecoder))
}

// generate script for checking transfer memo
func generateTransferMemoCheckScript(am auth.AccountKeeper, txDecoder sdk.TxDecoder) auth.Script {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Error {
		if !sdk.IsUpgrade(upgrade.BEP12) {
			return nil
		}

		sendMsg, ok := msg.(bank.MsgSend)
		if !ok {
			return nil
		}

		if txDecoder== nil {
			return nil
		}
		var err error
		var tx sdk.Tx
		var stdTx auth.StdTx
		for _, out := range sendMsg.Outputs {
			if isFlagEnabled(ctx, am, out.Address, TransferMemoCheckerFlag) {
				// Avoid tx decoding if no account enable this flag
				if tx == nil {
					tx, err = txDecoder(ctx.TxBytes())
					if err != nil {
						return sdk.ErrTxDecode(err.Error())
					}
					stdTx, ok = tx.(auth.StdTx)
					if !ok {
						return sdk.ErrInternal("tx must be StdTx")
					}
				}
				if len(stdTx.Memo) == 0 {
					return sdk.ErrInvalidTxMemo("receiver requires non-empty memo in transfer transaction")
				}
				for index := 0; index < len(stdTx.Memo); index++ {
					if stdTx.Memo[index] > '9' || stdTx.Memo[index] < '0' {
						return sdk.ErrInvalidTxMemo("receiver requires that memo should not contains non-digital letters")
					}
				}
			}
		}
		return nil
	}
}

func isFlagEnabled(ctx sdk.Context, am auth.AccountKeeper, addr sdk.AccAddress, targetFlag uint64) bool {
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		return false
	}
	account, ok := acc.(cmntypes.NamedAccount)
	if !ok {
		return false
	}
	if account.GetFlags() & targetFlag == 0 {
		return false
	}
	return true
}
