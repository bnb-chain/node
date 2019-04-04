package upgrade

import sdk "github.com/cosmos/cosmos-sdk/types"

var Mgr = sdk.UpgradeMgr

// prefix for the upgrade name
// bugfix: fix
// improvement: (maybe bip ?)
const FixOrderSeqInPriceLevelName = "fixOrderSeqInPriceLevel"
const FixDropFilledOrderSeqName = "fixDropFilledOrderSeq"
const AddFeeTypeForStakeTxName = "addFeeTypeForStakeTx"

func FixOrderSeqInPriceLevel(before func(), in func(), after func()) {
	sdk.Upgrade(FixOrderSeqInPriceLevelName, before, in, after)
}

func FixDropFilledOrderSeq(before func(), after func()) {
	sdk.Upgrade(FixDropFilledOrderSeqName, before, nil, after)
}
