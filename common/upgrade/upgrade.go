package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bep ?)
const BEP6 = "BEP6"   // https://github.com/binance-chain/BEPs/pull/6
const BEP9 = "BEP9"   // https://github.com/binance-chain/BEPs/pull/9
const BEP10 = "BEP10" // https://github.com/binance-chain/BEPs/pull/10

func UpgradeBEP10(before func(), after func()) {
	sdk.Upgrade(BEP10, before, nil, after)
}
