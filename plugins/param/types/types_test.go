package types

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/binance-chain/node/wire"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var testScParams = `[ { "type": "params/StakeParams", "value": { "Params": { "unbonding_time": "604800000000000", "max_validators": 11, "bond_denom": "BNB", "min_self_delegation": "5000000000000", "min_delegation_change": "100000000" } } }, { "type": "params/SlashParams", "value": { "Params": { "max_evidence_age": "259200000000000", "signed_blocks_window": "0", "min_signed_per_window": "0", "double_sign_unbond_duration": "9223372036854775807", "downtime_unbond_duration": "172800000000000", "too_low_del_unbond_duration": "86400000000000", "slash_fraction_double_sign": "0", "slash_fraction_downtime": "0", "double_sign_slash_amount": "1000000000000", "downtime_slash_amount": "5000000000", "submitter_reward": "100000000000", "downtime_slash_fee": "1000000000" } } }, { "type": "params/OracleParams", "value": { "Params": { "ConsensusNeeded": "70000000" } } } ]`

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
		{cp: generatSCParamChange(&OracleParams{Params: types.Params{ConsensusNeeded: sdk.NewDecWithPrec(7, 1)}}, 2), expectError: false},
		{cp: generatSCParamChange(&OracleParams{Params: types.Params{ConsensusNeeded: sdk.NewDecWithPrec(7, 0)}}, 2), expectError: true},
		{cp: generatSCParamChange(&OracleParams{Params: types.Params{ConsensusNeeded: sdk.ZeroDec()}}, 2), expectError: true},
		{cp: generatSCParamChange(&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5}}, 0), expectError: false},
		{cp: generatSCParamChange(&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB1", MinSelfDelegation: 100e8, MinDelegationChange: 1e5}}, 0), expectError: true},
		{cp: generatSCParamChange(&StakeParams{Params: stake.Params{UnbondingTime: 1 * time.Second, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5}}, 0), expectError: true},
		{cp: generatSCParamChange(&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 0, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5}}, 0), expectError: true},
		{cp: generatSCParamChange(&StakeParams{Params: stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 1e7, MinDelegationChange: 1e5}}, 0), expectError: true},
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

func generatSCParamChange(s SCParam, idx int) SCChangeParams {
	iScPrams := make([]SCParam, 0)
	cdc := amino.NewCodec()
	testRegisterWire(cdc)
	cdc.UnmarshalJSON([]byte(testScParams), &iScPrams)
	iScPrams[idx] = s
	return SCChangeParams{SCParams: iScPrams, Description: "test"}
}

// Register concrete types on wire codec
func testRegisterWire(cdc *wire.Codec) {
	cdc.RegisterInterface((*SCParam)(nil), nil)
	cdc.RegisterConcrete(&OracleParams{}, "params/OracleParams", nil)
	cdc.RegisterConcrete(&StakeParams{}, "params/StakeParams", nil)
	cdc.RegisterConcrete(&SlashParams{}, "params/SlashParams", nil)
}
