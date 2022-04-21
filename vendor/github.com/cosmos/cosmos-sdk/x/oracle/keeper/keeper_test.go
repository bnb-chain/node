package keeper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

const (
	TestID                     = "oracleID"
	AlternateTestID            = "altOracleID"
	TestString                 = "{value: 5}"
	AlternateTestString        = "{value: 7}"
	AnotherAlternateTestString = "{value: 9}"
)

var (
	pubkeys = []crypto.PubKey{ed25519.GenPrivKey().PubKey(), ed25519.GenPrivKey().PubKey(), ed25519.GenPrivKey().PubKey()}

	testDescription   = stake.NewDescription("T", "E", "S", "T")
	testCommissionMsg = stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())
)

func createValidators(t *testing.T, stakeHandler sdk.Handler, ctx sdk.Context, addrs []sdk.ValAddress, coinAmt []int64) {
	require.True(t, len(addrs) <= len(pubkeys), "Not enough pubkeys specified at top of file.")

	for i := 0; i < len(addrs); i++ {
		valCreateMsg := stake.NewMsgCreateValidator(
			addrs[i], pubkeys[i], sdk.NewCoin(gov.DefaultDepositDenom, coinAmt[i]), testDescription, testCommissionMsg,
		)

		res := stakeHandler(ctx, valCreateMsg)
		require.True(t, res.IsOK())
	}
}

func TestCreateGetProphecy(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 3)

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs))
	for i, addr := range addrs {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 5, 5})
	stake.EndBlocker(ctx, sk)

	keeper.SetParams(ctx, types.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)})

	validator1 := valAddrs[0]
	oracleClaim := types.NewClaim(TestID, validator1, TestString)
	prop, err := keeper.ProcessClaim(ctx, oracleClaim)

	require.NoError(t, err)
	require.Equal(t, prop.Status.Text, types.PendingStatusText)

	//Test bad Creation with blank id
	oracleClaim = types.NewClaim("", validator1, TestString)
	prop, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.Error(t, err)

	//Test bad Creation with blank claim
	oracleClaim = types.NewClaim(TestID, validator1, "")
	prop, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.Error(t, err)

	//Test retrieval
	prophecy, found := keeper.GetProphecy(ctx, TestID)
	require.True(t, found)
	require.Equal(t, prophecy.ID, TestID)
	require.Equal(t, prophecy.Status.Text, types.PendingStatusText)
	require.Equal(t, prophecy.ClaimValidators[TestString][0], validator1)
	require.Equal(t, prophecy.ValidatorClaims[validator1.String()], TestString)
}

func TestBadMsgs(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 3)

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs))
	for i, addr := range addrs {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 5, 5})
	stake.EndBlocker(ctx, sk)
	keeper.SetParams(ctx, types.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)})

	validator1Pow3 := valAddrs[0]

	//Test empty claim
	oracleClaim := types.NewClaim(TestID, validator1Pow3, "")
	prop, err := keeper.ProcessClaim(ctx, oracleClaim)
	require.Error(t, err)
	require.Equal(t, prop.Status.FinalClaim, "")
	require.True(t, strings.Contains(err.Error(), "claim cannot be empty string"))

	//Test normal Creation
	oracleClaim = types.NewClaim(TestID, validator1Pow3, TestString)
	prop, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, prop.Status.Text, types.PendingStatusText)
}

func TestSuccessfulProphecy(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 3)

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs))
	for i, addr := range addrs {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 5, 5})
	stake.EndBlocker(ctx, sk)
	keeper.SetParams(ctx, types.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)})

	validator1Pow3 := valAddrs[0]
	validator2Pow3 := valAddrs[1]
	validator3Pow4 := valAddrs[2]

	//Test first claim
	oracleClaim := types.NewClaim(TestID, validator1Pow3, TestString)
	status, err := keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.PendingStatusText)

	//Test second claim completes and finalizes to success
	oracleClaim = types.NewClaim(TestID, validator2Pow3, TestString)
	status, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.SuccessStatusText)
	require.Equal(t, status.Status.FinalClaim, TestString)

	//Test third claim not possible
	oracleClaim = types.NewClaim(TestID, validator3Pow4, TestString)
	status, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "prophecy already finalized"))
}

func TestSuccessfulProphecyWithDisagreement(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 3)

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs))
	for i, addr := range addrs {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 5, 5})
	stake.EndBlocker(ctx, sk)
	keeper.SetParams(ctx, types.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)})

	validator1Pow3 := valAddrs[0]
	validator2Pow3 := valAddrs[1]
	validator3Pow4 := valAddrs[2]

	//Test first claim
	oracleClaim := types.NewClaim(TestID, validator1Pow3, TestString)
	status, err := keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.PendingStatusText)

	//Test second disagreeing claim processed fine
	oracleClaim = types.NewClaim(TestID, validator2Pow3, AlternateTestString)
	status, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.PendingStatusText)

	//Test third claim agrees and finalizes to success
	oracleClaim = types.NewClaim(TestID, validator3Pow4, TestString)
	status, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.SuccessStatusText)
	require.Equal(t, status.Status.FinalClaim, TestString)
}

func TestFailedProphecy(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 3)

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs))
	for i, addr := range addrs {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 5, 5})
	stake.EndBlocker(ctx, sk)
	keeper.SetParams(ctx, types.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)})

	validator1Pow3 := valAddrs[0]
	validator2Pow3 := valAddrs[1]
	validator3Pow4 := valAddrs[2]

	//Test first claim
	oracleClaim := types.NewClaim(TestID, validator1Pow3, TestString)
	status, err := keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.PendingStatusText)

	//Test second disagreeing claim processed fine
	oracleClaim = types.NewClaim(TestID, validator2Pow3, AlternateTestString)
	status, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.PendingStatusText)
	require.Equal(t, status.Status.FinalClaim, "")

	//Test third disagreeing claim processed fine and prophecy fails
	oracleClaim = types.NewClaim(TestID, validator3Pow4, AnotherAlternateTestString)
	status, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.FailedStatusText)
	require.Equal(t, status.Status.FinalClaim, "")
}

func TestPowerOverrule(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 3)

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs))
	for i, addr := range addrs {
		valAddrs[i] = sdk.ValAddress(addr)
	}
	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 20, 5})
	stake.EndBlocker(ctx, sk)
	keeper.SetParams(ctx, types.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)})

	validator1Pow3 := valAddrs[0]
	validator2Pow7 := valAddrs[1]

	//Test first claim
	oracleClaim := types.NewClaim(TestID, validator1Pow3, TestString)
	status, err := keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.PendingStatusText)

	//Test second disagreeing claim processed fine and finalized to its bytes
	oracleClaim = types.NewClaim(TestID, validator2Pow7, AlternateTestString)
	status, err = keeper.ProcessClaim(ctx, oracleClaim)
	require.NoError(t, err)
	require.Equal(t, status.Status.Text, types.SuccessStatusText)
	require.Equal(t, status.Status.FinalClaim, AlternateTestString)
}

func TestNonValidator(t *testing.T) {
	//Test multiple prophecies running in parallel work fine as expected
	mapp, _, keeper, _, addrs, _, _ := getMockApp(t, 3)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	//Test claim on first id with first validator
	oracleClaim := types.NewClaim(TestID, sdk.ValAddress(addrs[0]), TestString)
	_, err := keeper.ProcessClaim(ctx, oracleClaim)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "claim must be made by actively bonded validator"))
}
