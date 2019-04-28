package types

import (
	"testing"

	"github.com/binance-chain/node/common/types"
	"github.com/stretchr/testify/assert"
)

func TestFixedFeeParamTypeCheck(t *testing.T) {
	testCases := []struct {
		fp          FixedFeeParams
		expectError bool
	}{
		{FixedFeeParams{"send", 0, types.FeeForProposer}, true},
		{FixedFeeParams{"submit_proposal", 0, types.FeeForProposer}, false},
		{FixedFeeParams{"remove_validator", 0, 0}, true},
		{FixedFeeParams{"tokensBurn", -1, types.FeeForProposer}, true},
		{FixedFeeParams{"tokensBurn", 100, types.FeeForProposer}, false},
	}
	for _, testCase := range testCases {
		err := testCase.fp.Check()
		if testCase.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestTransferFeeParamTypeCheck(t *testing.T) {
	testCases := []struct {
		fp          TransferFeeParam
		expectError bool
	}{
		{TransferFeeParam{FixedFeeParams{"send", 100, types.FeeForProposer}, 1, 2}, false},
		{TransferFeeParam{FixedFeeParams{"wrong type", 100, types.FeeForProposer}, 1, 2}, true},
		{TransferFeeParam{FixedFeeParams{"send", -1, types.FeeForProposer}, 1, 2}, true},
		{TransferFeeParam{FixedFeeParams{"send", 100, types.FeeForProposer}, 1, 1}, true},
	}
	for _, testCase := range testCases {
		err := testCase.fp.Check()
		if testCase.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestDexFeeParamTypeCheck(t *testing.T) {
	testCases := []struct {
		fp          DexFeeParam
		expectError bool
	}{
		{DexFeeParam{[]DexFeeField{{"ExpireFee", 1000}}}, false},
		{DexFeeParam{[]DexFeeField{{"ExpireFee", -1}}}, true},
	}
	for _, testCase := range testCases {
		err := testCase.fp.Check()
		if testCase.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestFeeChangeParamsCheck(t *testing.T) {
	testCases := []struct {
		fp          FeeChangeParams
		expectError bool
	}{
		{FeeChangeParams{FeeParams: []FeeParam{&DexFeeParam{[]DexFeeField{{"ExpireFee", 1000}}}, &TransferFeeParam{FixedFeeParams{"send", 100, types.FeeForProposer}, 1, 2}}}, false},
		{FeeChangeParams{FeeParams: []FeeParam{&DexFeeParam{[]DexFeeField{{"ExpireFee", 1000}}}, &FixedFeeParams{"send", 100, types.FeeForProposer}}}, true},
		{FeeChangeParams{FeeParams: []FeeParam{&DexFeeParam{[]DexFeeField{{"ExpireFee", 1000}}}, &DexFeeParam{[]DexFeeField{{"ExpireFee", 1000}}}}}, true},
	}
	for _, testCase := range testCases {
		err := testCase.fp.Check()
		if testCase.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
