package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bep ?)
const (
	// Galileo Upgrade
	BEP6  = "BEP6"   // https://github.com/bnb-chain/BEPs/pull/6
	BEP9  = sdk.BEP9 // https://github.com/bnb-chain/BEPs/pull/9
	BEP10 = "BEP10"  // https://github.com/bnb-chain/BEPs/pull/10
	BEP19 = "BEP19"  // https://github.com/bnb-chain/BEPs/pull/19  match engine revision
	// Hubble Upgrade
	BEP12 = sdk.BEP12 // https://github.com/bnb-chain/BEPs/pull/17
	// Archimedes Upgrade
	BEP3 = sdk.BEP3 // https://github.com/bnb-chain/BEPs/pull/30
	// Heisenberg Upgrade
	FixSignBytesOverflow = sdk.FixSignBytesOverflow
	LotSizeOptimization  = "LotSizeOptimization"
	ListingRuleUpgrade   = "ListingRuleUpgrade" // Remove restriction that only the owner of base asset can list trading pair
	FixZeroBalance       = "FixZeroBalance"

	LaunchBscUpgrade = sdk.LaunchBscUpgrade

	EnableAccountScriptsForCrossChainTransfer = "EnableAccountScriptsForCrossChainTransfer"

	//Nightingale upgrade
	BEP8  = sdk.BEP8 // https://github.com/bnb-chain/BEPs/pull/69 Mini token upgrade
	BEP67 = "BEP67"  // https://github.com/bnb-chain/BEPs/pull/67 Expiry time upgrade
	BEP70 = "BEP70"  // https://github.com/bnb-chain/BEPs/pull/70 BUSD Pair Upgrade

	BEP82             = sdk.BEP82 // https://github.com/bnb-chain/BEPs/pull/82
	BEP84             = "BEP84"   // https://github.com/bnb-chain/BEPs/pull/84 Mirror Sync Upgrade
	BEP87             = "BEP87"   // https://github.com/bnb-chain/BEPs/pull/87
	FixFailAckPackage = sdk.FixFailAckPackage

	BEP128 = sdk.BEP128 // https://github.com/bnb-chain/BEPs/pull/128 Staking reward distribution upgrade
	BEP151 = "BEP151"   // https://github.com/bnb-chain/BEPs/pull/151 Decommission Decentralized Exchange
	BEP153 = sdk.BEP153 // https://github.com/bnb-chain/BEPs/pull/153 Native Staking
	BEPHHH = sdk.BEPHHH // https://github.com/bnb-chain/BEPs/pull/HHH New Staking Mechanism
)

func UpgradeBEP10(before func(), after func()) {
	sdk.Upgrade(BEP10, before, nil, after)
}
