package types_test

import (
	"encoding/hex"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/oracle/types"
	fTypes "github.com/cosmos/cosmos-sdk/x/paramHub/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/common"
	"testing"
	"time"
)

var testScParams = `[{"type": "params/StakeParamSet","value": {"unbonding_time": "604800000000000","max_validators": 11,"bond_denom": "BNB","min_self_delegation": "5000000000000","min_delegation_change": "100000000", "reward_distribution_batch_size": "2000"}},{"type": "params/SlashParamSet","value": {"max_evidence_age": "259200000000000","signed_blocks_window": "0","min_signed_per_window": "0","double_sign_unbond_duration": "9223372036854775807","downtime_unbond_duration": "172800000000000","too_low_del_unbond_duration": "86400000000000","slash_fraction_double_sign": "0","slash_fraction_downtime": "0","double_sign_slash_amount": "1000000000000","downtime_slash_amount": "5000000000","submitter_reward": "100000000000","downtime_slash_fee": "1000000000"}},{"type": "params/OracleParamSet","value": {"ConsensusNeeded": "70000000"}},{"type": "params/IbcParamSet","value": {"relayer_fee": "1000000"}}]`

func TestFixedFeeParamTypeCheck(t *testing.T) {
	testCases := []struct {
		fp          fTypes.FixedFeeParams
		expectError bool
	}{
		{fTypes.FixedFeeParams{"send", 0, sdk.FeeForProposer}, true},
		{fTypes.FixedFeeParams{"submit_proposal", 0, sdk.FeeForProposer}, false},
		{fTypes.FixedFeeParams{"remove_validator", 0, 0}, true},
		{fTypes.FixedFeeParams{"tokensBurn", -1, sdk.FeeForProposer}, true},
		{fTypes.FixedFeeParams{"tokensBurn", 100, sdk.FeeForProposer}, false},
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
		fp          fTypes.TransferFeeParam
		expectError bool
	}{
		{fTypes.TransferFeeParam{fTypes.FixedFeeParams{"send", 100, sdk.FeeForProposer}, 1, 2}, false},
		{fTypes.TransferFeeParam{fTypes.FixedFeeParams{"wrong type", 100, sdk.FeeForProposer}, 1, 2}, true},
		{fTypes.TransferFeeParam{fTypes.FixedFeeParams{"send", -1, sdk.FeeForProposer}, 1, 2}, true},
		{fTypes.TransferFeeParam{fTypes.FixedFeeParams{"send", 100, sdk.FeeForProposer}, 1, 1}, true},
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
		fp          fTypes.DexFeeParam
		expectError bool
	}{
		{fTypes.DexFeeParam{[]fTypes.DexFeeField{{"ExpireFee", 1000}}}, false},
		{fTypes.DexFeeParam{[]fTypes.DexFeeField{{"ExpireFee", -1}}}, true},
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
		fp          fTypes.FeeChangeParams
		expectError bool
	}{
		{fTypes.FeeChangeParams{FeeParams: []fTypes.FeeParam{&fTypes.DexFeeParam{[]fTypes.DexFeeField{{"ExpireFee", 1000}}}, &fTypes.TransferFeeParam{fTypes.FixedFeeParams{"send", 100, sdk.FeeForProposer}, 1, 2}}}, false},
		{fTypes.FeeChangeParams{FeeParams: []fTypes.FeeParam{&fTypes.DexFeeParam{[]fTypes.DexFeeField{{"ExpireFee", 1000}}}, &fTypes.FixedFeeParams{"send", 100, sdk.FeeForProposer}}}, true},
		{fTypes.FeeChangeParams{FeeParams: []fTypes.FeeParam{&fTypes.DexFeeParam{[]fTypes.DexFeeField{{"ExpireFee", 1000}}}, &fTypes.DexFeeParam{[]fTypes.DexFeeField{{"ExpireFee", 1000}}}}}, true},
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
		cp          fTypes.CSCParamChange
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

func TestSCParamCheck(t *testing.T) {
	type TestCase struct {
		cp          fTypes.SCChangeParams
		expectError bool
	}
	testcases := []TestCase{
		{cp: generatSCParamChange(&types.Params{ConsensusNeeded: sdk.NewDecWithPrec(7, 1)}, 2), expectError: false},
		{cp: generatSCParamChange(&types.Params{ConsensusNeeded: sdk.NewDecWithPrec(7, 0)}, 2), expectError: true},
		{cp: generatSCParamChange(&types.Params{ConsensusNeeded: sdk.ZeroDec()}, 2), expectError: true},
		{cp: generatSCParamChange(&stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5, RewardDistributionBatchSize: 1000}, 0), expectError: false},
		{cp: generatSCParamChange(&stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB1", MinSelfDelegation: 100e8, MinDelegationChange: 1e5, RewardDistributionBatchSize: 1000}, 0), expectError: true},
		{cp: generatSCParamChange(&stake.Params{UnbondingTime: 1 * time.Second, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5, RewardDistributionBatchSize: 2000}, 0), expectError: true},
		{cp: generatSCParamChange(&stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 0, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5, RewardDistributionBatchSize: 3000}, 0), expectError: true},
		{cp: generatSCParamChange(&stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 1e7, MinDelegationChange: 1e5, RewardDistributionBatchSize: 8000}, 0), expectError: true},
		{cp: generatSCParamChange(&stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5, RewardDistributionBatchSize: 10}, 0), expectError: true},
		{cp: generatSCParamChange(&stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "BNB", MinSelfDelegation: 100e8, MinDelegationChange: 1e5, RewardDistributionBatchSize: 5010}, 0), expectError: true},
		{cp: fTypes.SCChangeParams{SCParams: []fTypes.SCParam{nil}}, expectError: true},
		{cp: fTypes.SCChangeParams{SCParams: []fTypes.SCParam{}}, expectError: true},
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

func generatCSCParamChange() fTypes.CSCParamChange {
	return fTypes.CSCParamChange{
		Key:    common.RandStr(common.RandIntn(255) + 1),
		Value:  hex.EncodeToString(common.RandBytes(common.RandIntn(255) + 1)),
		Target: hex.EncodeToString(common.RandBytes(20)),
	}
}

func generatSCParamChange(s fTypes.SCParam, idx int) fTypes.SCChangeParams {
	iScPrams := make([]fTypes.SCParam, 0)
	cdc := amino.NewCodec()
	testRegisterWire(cdc)
	cdc.UnmarshalJSON([]byte(testScParams), &iScPrams)
	iScPrams[idx] = s
	return fTypes.SCChangeParams{SCParams: iScPrams, Description: "test"}
}

// Register concrete types on wire codec
func testRegisterWire(cdc *amino.Codec) {
	cdc.RegisterInterface((*fTypes.SCParam)(nil), nil)
	cdc.RegisterConcrete(&types.Params{}, "params/OracleParamSet", nil)
	cdc.RegisterConcrete(&stake.Params{}, "params/StakeParamSet", nil)
	cdc.RegisterConcrete(&slashing.Params{}, "params/SlashParamSet", nil)
	cdc.RegisterConcrete(&ibc.Params{}, "params/IbcParamSet", nil)
}
