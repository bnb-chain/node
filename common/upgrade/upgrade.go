package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bip ?)
const FixOrderSeqInPriceLevelName = "fixOrderSeqInPriceLevel"
const FixDropFilledOrderSeqName = "fixDropFilledOrderSeq"

func init()  {
	Mgr.AddUpgradeHeight(FixOrderSeqInPriceLevelName, 2855000)
	Mgr.AddUpgradeHeight(FixDropFilledOrderSeqName, 2855000)
}

func Upgrade(name string, before func(), in func(), after func()) {
	if sdk.IsUpgradeHeight(name) {
		if in != nil {
			in()
		}
	} else if sdk.IsUpgrade(name) {
		if after != nil {
			after()
		}
	} else {
		if before != nil {
			before()
		}
	}
}

func FixOrderSeqInPriceLevel(before func(), in func(), after func()) {
	Upgrade(FixOrderSeqInPriceLevelName, before, in, after)
}

func FixDropFilledOrderSeq(before func(), after func()) {
	Upgrade(FixDropFilledOrderSeqName, before, nil, after)
}
