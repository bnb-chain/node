package types

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/tendermint/libs/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

func TestFixedFeeParamTypeCheck(t *testing.T) {
	testCases := []struct {
		fp          FixedFeeParams
		expectError bool
	}{
		{FixedFeeParams{"send", 0, sdk.FeeForProposer}, true},
		{FixedFeeParams{"submit_proposal", 0, sdk.FeeForProposer}, false},
		{FixedFeeParams{"remove_validator", 0, 0}, true},
		{FixedFeeParams{"tokensBurn", -1, sdk.FeeForProposer}, true},
		{FixedFeeParams{"tokensBurn", 100, sdk.FeeForProposer}, false},
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
		{TransferFeeParam{FixedFeeParams{"send", 100, sdk.FeeForProposer}, 1, 2}, false},
		{TransferFeeParam{FixedFeeParams{"wrong type", 100, sdk.FeeForProposer}, 1, 2}, true},
		{TransferFeeParam{FixedFeeParams{"send", -1, sdk.FeeForProposer}, 1, 2}, true},
		{TransferFeeParam{FixedFeeParams{"send", 100, sdk.FeeForProposer}, 1, 1}, true},
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
		{FeeChangeParams{FeeParams: []FeeParam{&DexFeeParam{[]DexFeeField{{"ExpireFee", 1000}}}, &TransferFeeParam{FixedFeeParams{"send", 100, sdk.FeeForProposer}, 1, 2}}}, false},
		{FeeChangeParams{FeeParams: []FeeParam{&DexFeeParam{[]DexFeeField{{"ExpireFee", 1000}}}, &FixedFeeParams{"send", 100, sdk.FeeForProposer}}}, true},
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

func TestCSCParamChangeCheck(t *testing.T) {
	type TestCase struct {
		cp          CSCParamChange
		expectError bool
	}
	testcases := make([]TestCase, 0, 100)
	for i := 0; i < 100; i++ {
		testcases = append(testcases, TestCase{cp: generatCSCParamChange(), expectError: false})
	}
	testcases[91].cp.Key = common.RandStr(255)
	testcases[92].cp.Value = hex.EncodeToString(common.RandBytes(255))

	// empty key
	testcases[93].cp.Key = ""
	testcases[93].expectError = true
	//key length exceed 255
	testcases[94].cp.Key = common.RandStr(256)
	testcases[94].expectError = true
	// empty value
	testcases[95].cp.Value = hex.EncodeToString([]byte{})
	testcases[95].expectError = true
	//value length exceed 255
	testcases[96].cp.Value = hex.EncodeToString(common.RandBytes(256))
	testcases[96].expectError = true
	// empty target
	testcases[97].cp.Target = hex.EncodeToString([]byte{})
	testcases[97].expectError = true
	//target length not 20
	testcases[98].cp.Target = hex.EncodeToString(common.RandBytes(19))
	testcases[98].expectError = true
	//target length not 20
	testcases[99].cp.Target = hex.EncodeToString(common.RandBytes(21))
	testcases[99].expectError = true

	for _, c := range testcases {
		err := c.cp.Check()
		if c.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

}

func TestCSCParamChangeSerialize(t *testing.T) {
	for i := 0; i < 1; i++ {
		cscParam := generatCSCParamChange()
		cscParam.Check()
		bz := cscParam.Serialize()
		assert.Equal(t, bz[0], byte(0x00))
		keyLength := int(bz[1])

		key := bz[2 : 2+keyLength]
		valLength := int(bz[2+keyLength])

		val := bz[3+keyLength : 3+keyLength+valLength]
		target := bz[3+keyLength+valLength : 3+keyLength+valLength+20]
		assert.Equal(t, len(bz), int(23+keyLength+valLength))
		assert.True(t, bytes.Compare(key, []byte(cscParam.Key)) == 0)
		assert.True(t, bytes.Compare(val, cscParam.ValueBytes) == 0)
		assert.True(t, bytes.Compare(target, cscParam.TargetBytes) == 0)
	}
}

func TestSCParamCheck(t *testing.T) {
	type TestCase struct {
		cp          SCChangeParams
		expectError bool
	}
	testcases := []TestCase{
		{cp: SCChangeParams{SCParams: []SCParam{&OracleParams{Params: types.Params{ConsensusNeeded: sdk.NewDecWithPrec(7, 1)}}}}, expectError: false},
		{cp: SCChangeParams{SCParams: []SCParam{&OracleParams{Params: types.Params{ConsensusNeeded: sdk.NewDecWithPrec(7, 0)}}}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{&OracleParams{Params: types.Params{ConsensusNeeded: sdk.ZeroDec()}}}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8}}}}, expectError: false},
		{cp: SCChangeParams{SCParams: []SCParam{&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB1", MinSelfDelegation: 100e8}}}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{&StakeParams{Params: stake.Params{UnbondingTime: 1 * time.Minute, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8}}}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 0, BondDenom: "BNB", MinSelfDelegation: 100e8}}}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 1e15}}}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{&OracleParams{Params: types.Params{ConsensusNeeded: sdk.NewDecWithPrec(7, 1)}},
			&OracleParams{Params: types.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)}}}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{nil}}, expectError: true},
		{cp: SCChangeParams{SCParams: []SCParam{}}, expectError: true},
	}

	for _, c := range testcases {
		err := c.cp.Check()
		if c.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

}

func generatCSCParamChange() CSCParamChange {
	return CSCParamChange{
		Key:    common.RandStr(common.RandIntn(255) + 1),
		Value:  hex.EncodeToString(common.RandBytes(common.RandIntn(255) + 1)),
		Target: hex.EncodeToString(common.RandBytes(20)),
	}
}
