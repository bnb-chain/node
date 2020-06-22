package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmmtypes "github.com/binance-chain/node/common/types"
)

func ConvertBSCAmountToBCAmount(contractDecimals int8, bscAmount sdk.Int) (int64, sdk.Error) {
	if contractDecimals == cmmtypes.TokenDecimals {
		return bscAmount.Int64(), nil
	}

	var bcAmount sdk.Int
	if contractDecimals >= cmmtypes.TokenDecimals {
		decimals := sdk.NewIntWithDecimal(1, int(contractDecimals-cmmtypes.TokenDecimals))
		if !bscAmount.Mod(decimals).IsZero() {
			return 0, ErrInvalidAmount(fmt.Sprintf("can't convert bep2(decimals: 8) bscAmount to ERC20(decimals: %d) bscAmount", contractDecimals))
		}
		bcAmount = bscAmount.Div(decimals)
	} else {
		decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-contractDecimals))
		bcAmount = bscAmount.Mul(decimals)
	}
	// since we only convert bsc amount in transfer out package to bc amount,
	// so it should not overflow
	return bcAmount.Int64(), nil
}

func ConvertBCAmountToBSCAmount(contractDecimals int8, bcAmount int64) (sdk.Int, sdk.Error) {
	if contractDecimals == cmmtypes.TokenDecimals {
		return sdk.NewInt(bcAmount), nil
	}

	var bscAmount sdk.Int
	if contractDecimals >= cmmtypes.TokenDecimals {
		decimals := sdk.NewIntWithDecimal(1, int(contractDecimals-cmmtypes.TokenDecimals))
		bscAmount = sdk.NewInt(bcAmount).Mul(decimals)
	} else {
		decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-contractDecimals))
		if !sdk.NewInt(bcAmount).Mod(decimals).IsZero() {
			return sdk.Int{}, ErrInvalidAmount(fmt.Sprintf("can't convert bep2(decimals: 8) amount to ERC20(decimals: %d) amount", contractDecimals))
		}
		bscAmount = sdk.NewInt(bcAmount).Div(decimals)
	}
	return bscAmount, nil
}
