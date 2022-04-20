package app

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/go-amino"
	abcicli "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkFees "github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/mock"
	otypes "github.com/cosmos/cosmos-sdk/x/oracle/types"
	pHub "github.com/cosmos/cosmos-sdk/x/paramHub"
	paramHub "github.com/cosmos/cosmos-sdk/x/paramHub/keeper"
	ptypes "github.com/cosmos/cosmos-sdk/x/paramHub/types"
	sTypes "github.com/cosmos/cosmos-sdk/x/sidechain/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/stake"

	ctypes "github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/plugins/dex"
	"github.com/bnb-chain/node/plugins/tokens"
	"github.com/bnb-chain/node/wire"
)

// util objects
var (
	memDB                             = dbm.NewMemDB()
	logger                            = log.NewTMLogger(os.Stdout)
	genAccs, addrs, pubKeys, privKeys = mock.CreateGenAccounts(4,
		sdk.Coins{sdk.NewCoin("BNB", 500000e8), sdk.NewCoin("BTC-000", 200e8)})
	testScParams = `[{"type": "params/StakeParamSet","value": {"unbonding_time": "604800000000000","max_validators": 11,"bond_denom": "BNB","min_self_delegation": "5000000000000","min_delegation_change": "100000000","reward_distribution_batch_size":"200"}},{"type": "params/SlashParamSet","value": {"max_evidence_age": "259200000000000","signed_blocks_window": "0","min_signed_per_window": "0","double_sign_unbond_duration": "9223372036854775807","downtime_unbond_duration": "172800000000000","too_low_del_unbond_duration": "86400000000000","slash_fraction_double_sign": "0","slash_fraction_downtime": "0","double_sign_slash_amount": "1000000000000","downtime_slash_amount": "5000000000","submitter_reward": "100000000000","downtime_slash_fee": "1000000000"}},{"type": "params/OracleParamSet","value": {"ConsensusNeeded": "70000000"}},{"type": "params/IbcParamSet","value": {"relayer_fee": "1000000"}}]`
	testClient   *TestClient
	testApp      *BinanceChain
)

func init() {
	ServerContext.UpgradeConfig.LaunchBscUpgradeHeight = 1
	testApp = NewBinanceChain(logger, memDB, os.Stdout)
	testClient = NewTestClient(testApp)
}

func TestCSCParamUpdatesSuccess(t *testing.T) {
	valAddr, ctx, accounts := setupTest()
	sideValAddr := accounts[0].GetAddress().Bytes()
	tNow := time.Now()

	ctx = UpdateContext(valAddr, ctx, 3, tNow.AddDate(0, 0, 1))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})

	cscParam := ptypes.CSCParamChange{
		Key:    "testa",
		Value:  hex.EncodeToString([]byte(hex.EncodeToString([]byte("testValue")))),
		Target: hex.EncodeToString(cmn.RandBytes(20)),
	}
	cscParam.Check()
	cscParamsBz, err := Codec.MarshalJSON(cscParam)
	proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(cscParamsBz), gov.ProposalTypeCSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
	res, err := testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
	fmt.Println(res)
	assert.NoError(t, err, "failed to submit side chain parameters change")

	voteMsg := gov.NewMsgSideChainVote(sideValAddr, 1, gov.OptionYes, "bsc")
	_, err = testClient.DeliverTxSync(&voteMsg, testApp.Codec)
	assert.NoError(t, err, "failed to vote side chain parameters change")
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	ctx = UpdateContext(valAddr, ctx, 4, tNow.AddDate(0, 0, 1).Add(5*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(5 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	packageBz, err := testApp.ibcKeeper.GetIBCPackage(ctx, "bsc", paramHub.ChannelName, uint64(0))
	assert.NoError(t, err)
	expectedBz, _ := rlp.EncodeToBytes(cscParam)
	assert.NoError(t, err)
	assert.True(t, bytes.Compare(expectedBz, packageBz[sTypes.PackageHeaderLength:]) == 0, "package bytes not equal")
}

func TestCSCParamUpdatesSequenceCorrect(t *testing.T) {
	valAddr, ctx, accounts := setupTest()
	sideValAddr := accounts[0].GetAddress().Bytes()
	tNow := time.Now()

	cscParams := []ptypes.CSCParamChange{
		{
			Key:    "testA",
			Value:  hex.EncodeToString([]byte(hex.EncodeToString([]byte("testValueA")))),
			Target: hex.EncodeToString(cmn.RandBytes(20)),
		},
		{
			Key:    "testB",
			Value:  hex.EncodeToString([]byte(hex.EncodeToString([]byte("testValueB")))),
			Target: hex.EncodeToString(cmn.RandBytes(20)),
		},
		{
			Key:    "testC",
			Value:  hex.EncodeToString([]byte(hex.EncodeToString([]byte("testValueC")))),
			Target: hex.EncodeToString(cmn.RandBytes(20)),
		},
	}
	for idx, c := range cscParams {
		c.Check()
		cscParams[idx] = c
	}

	ctx = UpdateContext(valAddr, ctx, 3, tNow.AddDate(0, 0, 1))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	for idx, cscParam := range cscParams {
		cscParamsBz, err := Codec.MarshalJSON(cscParam)
		proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(cscParamsBz), gov.ProposalTypeCSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
		_, err = testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
		assert.NoError(t, err, "failed to submit side chain parameters change")

		voteMsg := gov.NewMsgSideChainVote(sideValAddr, int64(idx+1), gov.OptionYes, "bsc")
		_, err = testClient.DeliverTxSync(&voteMsg, testApp.Codec)
		assert.NoError(t, err, "failed to vote side chain parameters change")
	}
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	ctx = UpdateContext(valAddr, ctx, 4, tNow.AddDate(0, 0, 1).Add(5*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(5 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	for idx, cscParam := range cscParams {
		packageBz, err := testApp.ibcKeeper.GetIBCPackage(ctx, "bsc", paramHub.ChannelName, uint64(idx))
		expectedBz, _ := rlp.EncodeToBytes(cscParam)
		assert.NoError(t, err)
		assert.True(t, bytes.Compare(expectedBz, packageBz[sTypes.PackageHeaderLength:]) == 0, "package bytes not equal")
	}

	// expire proposal
	ctx = UpdateContext(valAddr, ctx, 5, tNow.AddDate(0, 0, 1).Add(6*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(6 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	for _, cscParam := range cscParams {
		cscParamsBz, err := Codec.MarshalJSON(cscParam)
		proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(cscParamsBz), gov.ProposalTypeCSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
		_, err = testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
		assert.NoError(t, err, "failed to submit side chain parameters change")
	}
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	ctx = UpdateContext(valAddr, ctx, 6, tNow.AddDate(0, 0, 1).Add(10*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(10 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	packageBz, err := testApp.ibcKeeper.GetIBCPackage(ctx, "bsc", paramHub.ChannelName, uint64(3))
	assert.NoError(t, err)
	assert.True(t, len(packageBz) == 0, "write package unexpected")

	// still in order

	ctx = UpdateContext(valAddr, ctx, 7, tNow.AddDate(0, 0, 1).Add(12*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(12 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	for idx, cscParam := range cscParams {
		cscParamsBz, err := Codec.MarshalJSON(cscParam)
		proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(cscParamsBz), gov.ProposalTypeCSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
		_, err = testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
		assert.NoError(t, err, "failed to submit side chain parameters change")

		voteMsg := gov.NewMsgSideChainVote(sideValAddr, int64(idx+7), gov.OptionYes, "bsc")
		_, err = testClient.DeliverTxSync(&voteMsg, testApp.Codec)
		assert.NoError(t, err, "failed to vote side chain parameters change")
	}
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	ctx = UpdateContext(valAddr, ctx, 8, tNow.AddDate(0, 0, 1).Add(15*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(15 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	for idx, cscParam := range cscParams {
		packageBz, err := testApp.ibcKeeper.GetIBCPackage(ctx, "bsc", paramHub.ChannelName, uint64(idx+3))
		expectedBz, _ := rlp.EncodeToBytes(cscParam)
		assert.NoError(t, err)
		assert.True(t, bytes.Compare(expectedBz, packageBz[sTypes.PackageHeaderLength:]) == 0, "package bytes not equal")
	}
}

func TestSubmitCSCParamUpdatesFail(t *testing.T) {
	valAddr, ctx, accounts := setupTest()
	sideValAddr := accounts[0].GetAddress().Bytes()
	tNow := time.Now()

	ctx = UpdateContext(valAddr, ctx, 3, tNow.AddDate(0, 0, 1))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})

	cscParams := []ptypes.CSCParamChange{
		{
			Key:    "",
			Value:  hex.EncodeToString([]byte("testValue")),
			Target: hex.EncodeToString(cmn.RandBytes(20)),
		},
		{
			Key:    "testKey",
			Value:  "",
			Target: hex.EncodeToString(cmn.RandBytes(20)),
		},
		{
			Key:    "testKey",
			Value:  hex.EncodeToString([]byte("testValue")),
			Target: hex.EncodeToString(cmn.RandBytes(10)),
		},
		{
			Key:    cmn.RandStr(256),
			Value:  hex.EncodeToString([]byte("testValue")),
			Target: hex.EncodeToString(cmn.RandBytes(20)),
		},
	}

	for idx, c := range cscParams {
		c.Check()
		cscParams[idx] = c
	}

	for _, cscParam := range cscParams {
		cscParamsBz, err := Codec.MarshalJSON(cscParam)
		proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(cscParamsBz), gov.ProposalTypeCSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
		resp, err := testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
		assert.NoError(t, err, "failed to submit side chain parameters change")
		assert.True(t, strings.Contains(resp.Log, "Invalid proposal"))
	}
}

func TestSCParamUpdatesSuccess(t *testing.T) {
	valAddr, ctx, accounts := setupTest()
	sideValAddr := accounts[0].GetAddress().Bytes()
	tNow := time.Now()

	ctx = UpdateContext(valAddr, ctx, 3, tNow.AddDate(0, 0, 1))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})

	scParams := ptypes.SCChangeParams{
		SCParams: []ptypes.SCParam{
			generatSCParamChange(nil, 0).SCParams[0],
			generatSCParamChange(nil, 0).SCParams[1],
			&otypes.Params{ConsensusNeeded: sdk.NewDecWithPrec(9, 1)},
			generatSCParamChange(nil, 0).SCParams[3],
		}}
	scParamsBz, err := Codec.MarshalJSON(scParams)
	proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(scParamsBz), gov.ProposalTypeSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
	res, err := testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
	fmt.Println(res)
	assert.NoError(t, err, "failed to submit side chain parameters change")

	voteMsg := gov.NewMsgSideChainVote(sideValAddr, 1, gov.OptionYes, "bsc")
	_, err = testClient.DeliverTxSync(&voteMsg, testApp.Codec)
	assert.NoError(t, err, "failed to vote side chain parameters change")
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	// endblock
	ctx = UpdateContext(valAddr, ctx, 4, tNow.AddDate(0, 0, 1).Add(5*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(5 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	// breath block
	ctx = UpdateContext(valAddr, ctx, 5, tNow.AddDate(0, 0, 2).Add(5*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(5 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	p := testApp.oracleKeeper.GetConsensusNeeded(ctx)
	assert.True(t, p.Equal(sdk.NewDecWithPrec(9, 1)))
	storePrefix := testApp.scKeeper.GetSideChainStorePrefix(ctx, ServerContext.BscChainId)
	sideChainCtx := ctx.WithSideChainKeyPrefix(storePrefix)
	s := testApp.stakeKeeper.GetParams(sideChainCtx)
	assert.True(t, s.Equal(*(scParams.SCParams[0].(*stake.Params))))
}

func TestSCParamMultiUpdatesSuccess(t *testing.T) {
	valAddr, ctx, accounts := setupTest()
	sideValAddr := accounts[0].GetAddress().Bytes()
	tNow := time.Now()

	ctx = UpdateContext(valAddr, ctx, 3, tNow.AddDate(0, 0, 1))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})

	scParamses := []ptypes.SCChangeParams{
		generatSCParamChange(&otypes.Params{ConsensusNeeded: sdk.NewDecWithPrec(6, 1)}, 2),
		generatSCParamChange(&otypes.Params{ConsensusNeeded: sdk.NewDecWithPrec(8, 1)}, 2),
		generatSCParamChange(&otypes.Params{ConsensusNeeded: sdk.NewDecWithPrec(9, 1)}, 2),
	}
	for idx, scParams := range scParamses {
		scParamsBz, err := Codec.MarshalJSON(scParams)
		proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(scParamsBz), gov.ProposalTypeSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
		_, err = testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
		assert.NoError(t, err, "failed to submit side chain parameters change")

		voteMsg := gov.NewMsgSideChainVote(sideValAddr, int64(idx+1), gov.OptionYes, "bsc")
		_, err = testClient.DeliverTxSync(&voteMsg, testApp.Codec)
		assert.NoError(t, err, "failed to vote side chain parameters change")
	}
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	// endblock
	ctx = UpdateContext(valAddr, ctx, 4, tNow.AddDate(0, 0, 1).Add(5*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(5 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	// breath block
	ctx = UpdateContext(valAddr, ctx, 5, tNow.AddDate(0, 0, 2).Add(5*time.Second))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1).Add(5 * time.Second)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{Height: ctx.BlockHeader().Height})

	p := testApp.oracleKeeper.GetConsensusNeeded(ctx)
	assert.True(t, p.Equal(sdk.NewDecWithPrec(9, 1)))
}

func TestSCParamUpdatesFail(t *testing.T) {
	valAddr, ctx, accounts := setupTest()
	sideValAddr := accounts[0].GetAddress().Bytes()
	tNow := time.Now()

	ctx = UpdateContext(valAddr, ctx, 3, tNow.AddDate(0, 0, 1))
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, 1)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})

	scParamses := []ptypes.SCChangeParams{
		generatSCParamChange(&stake.Params{UnbondingTime: 24 * time.Hour, MaxValidators: 10, BondDenom: "", MinSelfDelegation: 100e8}, 0),
		generatSCParamChange(&otypes.Params{ConsensusNeeded: sdk.NewDecWithPrec(2, 0)}, 2),
		{SCParams: []ptypes.SCParam{
			nil,
		}},
		{SCParams: nil},
	}
	for _, scParams := range scParamses {
		scParamsBz, err := Codec.MarshalJSON(scParams)
		proposeMsg := gov.NewMsgSideChainSubmitProposal("testSideProposal", string(scParamsBz), gov.ProposalTypeSCParamsChange, sideValAddr, sdk.Coins{sdk.Coin{"BNB", 2000e8}}, time.Second, "bsc")
		res, err := testClient.DeliverTxSync(&proposeMsg, testApp.Codec)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(res.Log, "Invalid proposal"))
	}

}

// ===========  setup for test cases ====

func NewTestClient(a *BinanceChain) *TestClient {
	a.SetDeliverState(types.Header{})
	a.SetAnteHandler(newMockAnteHandler(a.Codec)) // clear AnteHandler to skip the signature verification step
	return &TestClient{abcicli.NewLocalClient(nil, a), MakeCodec()}
}

type TestClient struct {
	cl  abcicli.Client
	cdc *wire.Codec
}

func (tc *TestClient) DeliverTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseDeliverTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)

	return tc.cl.DeliverTxSync(abci.RequestDeliverTx{Tx: tx})
}

func newMockAnteHandler(cdc *wire.Codec) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, runTxMode sdk.RunTxMode) (sdk.Context, sdk.Result, bool) {
		msg := tx.GetMsgs()[0]
		fee := sdkFees.GetCalculator(msg.Type())(msg)

		if ctx.IsDeliverTx() {
			// add fee to pool, even it's free
			stdTx := tx.(auth.StdTx)
			txHash := cmn.HexBytes(tmhash.Sum(cdc.MustMarshalBinaryLengthPrefixed(stdTx))).String()
			sdkFees.Pool.AddFee(txHash, fee)
		}
		return ctx, sdk.Result{}, false
	}
}

func UpdateContext(addr crypto.Address, ctx sdk.Context, height int64, tNow time.Time) sdk.Context {
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: addr, Height: height, Time: tNow}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: addr, Power: 10}, SignedLastBlock: true},
	}).WithBlockHash([]byte("testhash"))

	testApp.DeliverState.Ctx = ctx
	return ctx
}

func setupTest() (crypto.Address, sdk.Context, []sdk.Account) {
	// for old match engine
	addr := secp256k1.GenPrivKey().PubKey().Address()
	accAddr := sdk.AccAddress(addr)
	baseAcc := auth.BaseAccount{Address: accAddr}
	genTokens := []tokens.GenesisToken{{"BNB", "BNB", 100000000e8, accAddr, false}}
	appAcc := &ctypes.AppAccount{baseAcc, "baseAcc", sdk.Coins(nil), sdk.Coins(nil), 0}
	genAccs := make([]GenesisAccount, 1)
	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genAccs[0] = NewGenesisAccount(appAcc, valAddr)
	genesisState := GenesisState{
		Tokens:       genTokens,
		Accounts:     genAccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: pHub.DefaultGenesisState,
	}
	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}
	testApp.SetCheckState(abci.Header{})
	testApp.InitChain(abci.RequestInitChain{
		Validators:    []abci.ValidatorUpdate{},
		AppStateBytes: stateBytes})
	// it is required in fee distribution during end block
	testApp.ValAddrCache.SetAccAddr(sdk.ConsAddress(valAddr), appAcc.Address)
	ctx := testApp.DeliverState.Ctx
	coins := sdk.Coins{sdk.NewCoin("BNB", 1e13)}
	var accs []sdk.Account
	for i := 0; i < 10; i++ {
		privKey := ed25519.GenPrivKey()
		pubKey := privKey.PubKey()
		addr := sdk.AccAddress(pubKey.Address())
		acc := &auth.BaseAccount{
			Address: addr,
			Coins:   coins,
		}
		appAcc := &ctypes.AppAccount{BaseAccount: *acc}
		if testApp.AccountKeeper.GetAccount(ctx, acc.GetAddress()) == nil {
			appAcc.BaseAccount.AccountNumber = testApp.AccountKeeper.GetNextAccountNumber(ctx)
		}
		testApp.AccountKeeper.SetAccount(ctx, appAcc)
		accs = append(accs, acc)
	}

	sideValAddr := accs[0].GetAddress().Bytes()
	tNow := time.Now()

	// empty first block
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -2)})
	ctx = UpdateContext(valAddr, ctx, 1, tNow.AddDate(0, 0, -2))
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	// second breath block to create side chain validator
	ctx = UpdateContext(valAddr, ctx, 2, tNow)
	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -2)})
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	msg := stake.NewMsgCreateSideChainValidator(sdk.ValAddress(sideValAddr), sdk.NewCoin("BNB", 5000000000000),
		stake.NewDescription("m", "i", "w", "d"),
		stake.NewCommissionMsg(sdk.NewDecWithPrec(1, 4), sdk.NewDecWithPrec(1, 4), sdk.NewDecWithPrec(1, 4)),
		"bsc", sideValAddr, sideValAddr)

	_, err = testClient.DeliverTxSync(&msg, testApp.Codec)
	if err != nil {
		panic(err)
	}
	testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	return valAddr, ctx, accs
}

func generatSCParamChange(s ptypes.SCParam, idx int) ptypes.SCChangeParams {
	iScPrams := make([]ptypes.SCParam, 0)
	cdc := amino.NewCodec()
	testRegisterWire(cdc)
	cdc.UnmarshalJSON([]byte(testScParams), &iScPrams)
	if s != nil {
		iScPrams[idx] = s
	}
	return ptypes.SCChangeParams{SCParams: iScPrams, Description: "test"}
}

// Register concrete types on wire codec
func testRegisterWire(cdc *wire.Codec) {
	cdc.RegisterInterface((*ptypes.SCParam)(nil), nil)
	cdc.RegisterConcrete(&ibc.Params{}, "params/IbcParamSet", nil)
	cdc.RegisterConcrete(&otypes.Params{}, "params/OracleParamSet", nil)
	cdc.RegisterConcrete(&slashing.Params{}, "params/SlashParamSet", nil)
	cdc.RegisterConcrete(&stake.Params{}, "params/StakeParamSet", nil)
}
