package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bip ?)

const (
	FixOrderSeqInPriceLevelName = "fixOrderSeqInPriceLevel"
	FixDropFilledOrderSeqName   = "fixDropFilledOrderSeq"
	FixLotSizeName              = "fixLotSize"
	FixOverflowsName            = "fixOverflows"
	AddFeeTypeForStakeTxName    = "addFeeTypeForStakeTx"
	FixOrderTimestampName       = "fixOrderTimestamp"
	UpgradeGovStrategy          = "UpgradeGovStrategy"
)

func FixOrderSeqInPriceLevel(before func(), in func(), after func()) {
	sdk.Upgrade(FixOrderSeqInPriceLevelName, before, in, after)
}

func FixDropFilledOrderSeq(before func(), after func()) {
	sdk.Upgrade(FixDropFilledOrderSeqName, before, nil, after)
}

func FixLotSize(before func(), after func()) {
	sdk.Upgrade(FixLotSizeName, before, nil, after)
}

func FixOverflows(before func(), after func()) {
	sdk.Upgrade(FixOverflowsName, before, nil, after)
}

func FixOrderTimestamp(before func(), after func()) {
	// deliberately not rebuild data here because rebuild means we need iterate all open orders
	// and update their timestamps
	sdk.Upgrade(FixOrderTimestampName, before, nil, after)
}

func ShouldRebuildGov() bool {
	upgradeHeight := Mgr.GetUpgradeHeight(sdk.UpgradeRestoreGovStorage)
	if Mgr.GetHeight() == (upgradeHeight - 1) {
		return true
	}
	return false
}
