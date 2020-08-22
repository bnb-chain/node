package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestConvertBSCAmountToBCAmount(t *testing.T) {
	tests := []struct {
		contractDecimals int8
		bscAmount        sdk.Int
		bcAmount         int64
		expectedError    bool
	}{
		{
			10,
			sdk.NewInt(88),
			0,
			true,
		}, {
			10,
			sdk.NewInt(1000),
			10,
			false,
		}, {
			8,
			sdk.NewInt(1000),
			1000,
			false,
		}, {
			7,
			sdk.NewInt(1000),
			10000,
			false,
		},
	}
	for i, test := range tests {
		bcAmount, err := ConvertBSCAmountToBCAmount(test.contractDecimals, test.bscAmount)
		if test.expectedError {
			require.NotNil(t, err, "test: %d should return error", i)
		} else {
			require.Equal(t, bcAmount, test.bcAmount)
		}
	}
}

func TestConvertBCAmountToBSCAmount(t *testing.T) {
	tests := []struct {
		contractDecimals int8
		bcAmount         int64
		bscAmount        sdk.Int
		expectedError    bool
	}{
		{
			10,
			10,
			sdk.NewInt(1000),
			false,
		}, {
			8,
			10,
			sdk.NewInt(10),
			false,
		}, {
			6,
			90,
			sdk.NewInt(0),
			true,
		}, {
			6,
			900,
			sdk.NewInt(9),
			false,
		},
	}
	for i, test := range tests {
		bscAmount, err := ConvertBCAmountToBSCAmount(test.contractDecimals, test.bcAmount)
		if test.expectedError {
			require.NotNil(t, err, "test: %d should return error", i)
		} else {
			require.Equal(t, true, bscAmount.Equal(test.bscAmount))
		}
	}
}
