package scripts

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/bnb-chain/node/common/upgrade"
)

func RegisterTransferMemoCheckScript(am auth.AccountKeeper) {
	msgType := bank.MsgSend{}.Type()
	sdk.RegisterScripts(msgType, generateTransferMemoCheckScript(am))
}

// generate script for checking transfer memo
func generateTransferMemoCheckScript(am auth.AccountKeeper) sdk.Script {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Error {
		if !sdk.IsUpgrade(upgrade.BEP12) {
			return nil
		}

		sendMsg, ok := msg.(bank.MsgSend)
		if !ok {
			return nil
		}

		tx := ctx.Tx()
		if tx == nil {
			return sdk.ErrInternal("missing Tx in context")
		}
		stdTx, ok := tx.(auth.StdTx)
		if !ok {
			return sdk.ErrInternal("tx must be StdTx")
		}
		for _, out := range sendMsg.Outputs {
			if isFlagEnabled(ctx, am, out.Address, TransferMemoCheckerFlag) {
				if len(stdTx.Memo) == 0 {
					return sdk.ErrInvalidTxMemo("receiver requires non-empty memo in transfer transaction")
				}
				for index := 0; index < len(stdTx.Memo); index++ {
					if stdTx.Memo[index] > '9' || stdTx.Memo[index] < '0' {
						return sdk.ErrInvalidTxMemo("The receiver requires the memo contains only digits.")
					}
				}
			}
		}
		return nil
	}
}
