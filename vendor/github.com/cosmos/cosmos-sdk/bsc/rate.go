package bsc

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	BNBDecimalOnBC  = 8
	BNBDecimalOnBSC = 18
)

func ConvertBCAmountToBSCAmount(bcAmount int64) *big.Int {
	decimals := sdk.NewIntWithDecimal(1, int(BNBDecimalOnBSC-BNBDecimalOnBC))
	bscAmount := sdk.NewInt(bcAmount).Mul(decimals)
	return bscAmount.BigInt()
}
