package upgrade

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

	BEP126                      = sdk.BEP126 // https://github.com/bnb-chain/BEPs/pull/126 Fast finality upgrade
	BEP128                      = sdk.BEP128 // https://github.com/bnb-chain/BEPs/pull/128 Staking reward distribution upgrade
	BEP151                      = "BEP151"   // https://github.com/bnb-chain/BEPs/pull/151 Decommission Decentralized Exchange
	BEP153                      = sdk.BEP153 // https://github.com/bnb-chain/BEPs/pull/153 Native Staking
	BEP159                      = sdk.BEP159 // https://github.com/bnb-chain/BEPs/pull/159 New Staking Mechanism
	BEP159Phase2                = sdk.BEP159Phase2
	LimitConsAddrUpdateInterval = sdk.LimitConsAddrUpdateInterval
	BEP171                      = sdk.BEP171 // https://github.com/bnb-chain/BEPs/pull/171 Security Enhancement for Cross-Chain Module
	BEP173                      = sdk.BEP173 // https://github.com/bnb-chain/BEPs/pull/173 Text Proposal
	FixDoubleSignChainId        = sdk.FixDoubleSignChainId
	BEP255                      = sdk.BEP255 // https://github.com/bnb-chain/BEPs/pull/255 Asset Reconciliation for Security Enhancement
)

func UpgradeBEP10(before func(), after func()) {
	sdk.Upgrade(BEP10, before, nil, after)
}
