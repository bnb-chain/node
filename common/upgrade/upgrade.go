package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bip ?)
const FixOrderSeqInPriceLevelName = "fixOrderSeqInPriceLevel"
const FixDropFilledOrderSeqName = "fixDropFilledOrderSeq"
const FixOrderTimestampName = "fixOrderTimestamp"

func FixOrderSeqInPriceLevel(before func(), in func(), after func()) {
	sdk.Upgrade(FixOrderSeqInPriceLevelName, before, in, after)
}

func FixDropFilledOrderSeq(before func(), after func()) {
	sdk.Upgrade(FixDropFilledOrderSeqName, before, nil, after)
}

func FixOrderTimestamp(before func(), after func()) {
	// deliberately not rebuild data here because rebuild means we need iterate all open orders
	// and update their timestamps
	sdk.Upgrade(FixOrderTimestampName, before, nil, after)
}
