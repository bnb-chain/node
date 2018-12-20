package admin

import (
	"fmt"

	"github.com/BiJie/BinanceChain/common/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

var AllowedTxs = map[runtime.Mode][]string{
	runtime.TransferOnlyMode: {bank.MsgSend{}.Type()},
	runtime.RecoverOnlyMode:  {},
}

func TxNotAllowedError() sdk.Error {
	return sdk.ErrInternal(fmt.Sprintf("The tx is not allowed, RunningMode: %v", runtime.RunningMode))
}

func IsTxAllowed(tx sdk.Tx) bool {
	if runtime.RunningMode == runtime.NormalMode {
		return true
	}

	for _, msg := range tx.GetMsgs() {
		if !isMsgAllowed(msg) {
			return false
		}
	}
	return true
}

func isMsgAllowed(msg sdk.Msg) bool {
	for _, msgType := range AllowedTxs[runtime.RunningMode] {
		if msgType == msg.Type() {
			return true
		}
	}

	return false
}