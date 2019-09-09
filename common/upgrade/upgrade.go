package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: BEP

// Galileo Upgrade
const BEP6 = "BEP6"   // https://github.com/binance-chain/BEPs/pull/6
const BEP9 = "BEP9"   // https://github.com/binance-chain/BEPs/pull/9
const BEP10 = "BEP10" // https://github.com/binance-chain/BEPs/pull/10
const BEP19 = "BEP19" // https://github.com/binance-chain/BEPs/pull/19  match engine revision

// Hubble Upgrade
const BEP12 = "BEP12" // https://github.com/binance-chain/BEPs/pull/17

// Archimedes Upgrade
const MakerTakerFee = "MakerTakerFee"

func UpgradeBEP10(before func(), after func()) {
	sdk.Upgrade(BEP10, before, nil, after)
}
