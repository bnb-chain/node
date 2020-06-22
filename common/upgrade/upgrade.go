package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bep ?)
const (
	// Galileo Upgrade
	BEP6  = "BEP6"   // https://github.com/binance-chain/BEPs/pull/6
	BEP9  = sdk.BEP9 // https://github.com/binance-chain/BEPs/pull/9
	BEP10 = "BEP10"  // https://github.com/binance-chain/BEPs/pull/10
	BEP19 = "BEP19"  // https://github.com/binance-chain/BEPs/pull/19  match engine revision
	// Hubble Upgrade
	BEP12 = sdk.BEP12 // https://github.com/binance-chain/BEPs/pull/17
	// Archimedes Upgrade
	BEP3 = sdk.BEP3 // https://github.com/binance-chain/BEPs/pull/30
	// Heisenberg Upgrade
	FixSignBytesOverflow = sdk.FixSignBytesOverflow
	LotSizeOptimization  = "LotSizeOptimization"
	ListingRuleUpgrade   = "ListingRuleUpgrade" // Remove restriction that only the owner of base asset can list trading pair
	FixZeroBalance       = "FixZeroBalance"
	// TODO: add upgrade name
	LaunchBscUpgrade = sdk.LaunchBscUpgrade
)

func UpgradeBEP10(before func(), after func()) {
	sdk.Upgrade(BEP10, before, nil, after)
}
