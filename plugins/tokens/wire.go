package tokens

import (
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/plugins/tokens/timelock"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(issue.IssueMsg{}, "tokens/IssueMsg", nil)
	cdc.RegisterConcrete(issue.MintMsg{}, "tokens/MintMsg", nil)
	cdc.RegisterConcrete(burn.BurnMsg{}, "tokens/BurnMsg", nil)
	cdc.RegisterConcrete(freeze.FreezeMsg{}, "tokens/FreezeMsg", nil)
	cdc.RegisterConcrete(freeze.UnfreezeMsg{}, "tokens/UnfreezeMsg", nil)
	cdc.RegisterConcrete(timelock.TimeLockMsg{}, "tokens/TimeLockMsg", nil)
	cdc.RegisterConcrete(timelock.TimeUnlockMsg{}, "tokens/TimeUnlockMsg", nil)
	cdc.RegisterConcrete(timelock.TimeRelockMsg{}, "tokens/TimeRelockMsg", nil)
	cdc.RegisterConcrete(swap.HashTimerLockTransferMsg{}, "tokens/HashTimerLockTransferMsg", nil)
	cdc.RegisterConcrete(swap.DepositHashTimerLockMsg{}, "tokens/DepositHashTimerLockMsg", nil)
	cdc.RegisterConcrete(swap.ClaimHashTimerLockMsg{}, "tokens/ClaimHashTimerLockMsg", nil)
	cdc.RegisterConcrete(swap.RefundHashTimerLockMsg{}, "tokens/RefundHashTimerLockMsg", nil)
}
