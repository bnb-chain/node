package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bep ?)

const BEP6 = "BEP6"
const BEP9 = "BEP9"

func IsBEP9Upgrade() bool {
	return Mgr.GetHeight() >= Mgr.GetUpgradeHeight(BEP9)
}
