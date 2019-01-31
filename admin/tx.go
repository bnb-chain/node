package admin

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/runtime"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
)

var transferOnlyModeBlackList = []string{
	burn.BurnMsg{}.Type(),
	freeze.FreezeMsg{}.Type(),
	freeze.UnfreezeMsg{}.Type(),
	issue.IssueMsg{}.Type(),
	issue.MintMsg{}.Type(),
	order.NewOrderMsg{}.Type(),
	order.CancelOrderMsg{}.Type(),
}

var TxBlackList = map[runtime.Mode][]string{
	runtime.TransferOnlyMode: transferOnlyModeBlackList,
	runtime.RecoverOnlyMode:  append(transferOnlyModeBlackList, bank.MsgSend{}.Type()),
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
	for _, msgType := range TxBlackList[runtime.RunningMode] {
		if msgType == msg.Type() {
			return false
		}
	}

	return true
}
