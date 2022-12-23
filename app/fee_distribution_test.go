package app

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tidwall/gjson"

	"github.com/bnb-chain/node/app/pub"
	"github.com/bnb-chain/node/common/testutils"
	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/common/upgrade"
	"github.com/bnb-chain/node/wire"
)

const BREATHE_BLOCK_INTERVAL = 5

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore, accountKey *sdk.KVStoreKey) sdk.AccountCache {
	accountStore := ms.GetKVStore(accountKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

func setup() (am auth.AccountKeeper, valAddrCache *ValAddrCache, ctx sdk.Context, proposerAcc, valAcc1, valAcc2, valAcc3 sdk.Account) {
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	am = auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	valAddrCache = NewValAddrCache(stake.Keeper{})
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx = sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	// setup proposer and other validators
	_, proposerAcc = testutils.NewAccount(ctx, am, 100)
	_, valAcc1 = testutils.NewAccount(ctx, am, 100)
	_, valAcc2 = testutils.NewAccount(ctx, am, 100)
	_, valAcc3 = testutils.NewAccount(ctx, am, 100)
	proposerValAddr := ed25519.GenPrivKey().PubKey().Address()
	val1ValAddr := ed25519.GenPrivKey().PubKey().Address()
	val2ValAddr := ed25519.GenPrivKey().PubKey().Address()
	val3ValAddr := ed25519.GenPrivKey().PubKey().Address()

	valAddrCache.cache[string(proposerValAddr)] = proposerAcc.GetAddress()
	valAddrCache.cache[string(val1ValAddr)] = valAcc1.GetAddress()
	valAddrCache.cache[string(val2ValAddr)] = valAcc2.GetAddress()
	valAddrCache.cache[string(val3ValAddr)] = valAcc3.GetAddress()

	proposer := abci.Validator{Address: proposerValAddr, Power: 10}
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: proposerValAddr}).WithVoteInfos([]abci.VoteInfo{
		{Validator: proposer, SignedLastBlock: true},
		{Validator: abci.Validator{Address: val1ValAddr, Power: 10}, SignedLastBlock: true},
		{Validator: abci.Validator{Address: val2ValAddr, Power: 10}, SignedLastBlock: true},
		{Validator: abci.Validator{Address: val3ValAddr, Power: 10}, SignedLastBlock: true},
	})

	return
}

func checkBalance(t *testing.T, ctx sdk.Context, am auth.AccountKeeper, valAddrCache *ValAddrCache, balances []int64) {
	for i, voteInfo := range ctx.VoteInfos() {
		accAddr := valAddrCache.GetAccAddr(ctx, voteInfo.Validator.Address)
		valAcc := am.GetAccount(ctx, accAddr)
		require.Equal(t, balances[i], valAcc.GetCoins().AmountOf(types.NativeTokenSymbol))
	}
}

func TestNoFeeDistribution(t *testing.T) {
	// setup
	am, valAddrCache, ctx, _, _, _, _ := setup()
	fee := fees.Pool.BlockFees()
	require.True(t, true, fee.IsEmpty())

	blockFee := distributeFee(ctx, am, valAddrCache, true)
	fees.Pool.Clear()
	require.Equal(t, pub.BlockFee{0, "", nil}, blockFee)
	checkBalance(t, ctx, am, valAddrCache, []int64{100, 100, 100, 100})
}

func TestFeeDistribution2Proposer(t *testing.T) {
	// setup
	am, valAddrCache, ctx, proposerAcc, _, _, _ := setup()
	fees.Pool.AddAndCommitFee("DIST", sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, sdk.FeeForProposer))
	blockFee := distributeFee(ctx, am, valAddrCache, true)
	fees.Pool.Clear()
	require.Equal(t, pub.BlockFee{0, "BNB:10", []string{string(proposerAcc.GetAddress())}}, blockFee)
	checkBalance(t, ctx, am, valAddrCache, []int64{110, 100, 100, 100})
}

func TestFeeDistribution2AllValidators(t *testing.T) {
	// setup
	am, valAddrCache, ctx, proposerAcc, valAcc1, valAcc2, valAcc3 := setup()
	// fee amount can be divided evenly
	fees.Pool.AddAndCommitFee("DIST", sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 40)}, sdk.FeeForAll))
	blockFee := distributeFee(ctx, am, valAddrCache, true)
	// Notice: clean the pool after distributeFee
	fees.Pool.Clear()
	require.Equal(t, pub.BlockFee{0, "BNB:40", []string{string(proposerAcc.GetAddress()), string(valAcc1.GetAddress()), string(valAcc2.GetAddress()), string(valAcc3.GetAddress())}}, blockFee)
	checkBalance(t, ctx, am, valAddrCache, []int64{110, 110, 110, 110})

	// cannot be divided evenly
	fees.Pool.AddAndCommitFee("DIST", sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 50)}, sdk.FeeForAll))
	blockFee = distributeFee(ctx, am, valAddrCache, true)
	fees.Pool.Clear()
	require.Equal(t, pub.BlockFee{0, "BNB:50", []string{string(proposerAcc.GetAddress()), string(valAcc1.GetAddress()), string(valAcc2.GetAddress()), string(valAcc3.GetAddress())}}, blockFee)
	checkBalance(t, ctx, am, valAddrCache, []int64{124, 122, 122, 122})
}

type Account struct {
	Priv           crypto.PrivKey
	CryptoAddress  crypto.Address
	Address        sdk.AccAddress
	ValAddress     sdk.ValAddress
	BaseAccount    *auth.BaseAccount
	AppAccount     types.AppAccount
	GenesisAccount GenesisAccount
}

func GenAccounts(n int) (accounts []Account) {
	for i := 0; i < n; i++ {
		priv := ed25519.GenPrivKey()
		address := priv.PubKey().Address()
		accAddr := sdk.AccAddress(address)
		genCoin := sdk.NewCoin("BNB", 10e13)
		baseAccount := auth.BaseAccount{
			Address: accAddr,
			Coins:   sdk.Coins{genCoin},
		}
		appAcc := types.AppAccount{BaseAccount: baseAccount}
		genesisAccount := NewGenesisAccount(&appAcc, address)
		accounts = append(accounts, Account{
			Priv:           priv,
			CryptoAddress:  address,
			Address:        accAddr,
			BaseAccount:    &baseAccount,
			AppAccount:     appAcc,
			ValAddress:     sdk.ValAddress(address),
			GenesisAccount: genesisAccount,
		})
	}
	return
}

func setupTestForBEP159Test() (*BinanceChain, sdk.Context, []Account) {
	// config
	upgrade.Mgr.Reset()
	context := ServerContext
	ServerContext.BreatheBlockInterval = BREATHE_BLOCK_INTERVAL
	ServerContext.LaunchBscUpgradeHeight = 1
	ServerContext.BEP128Height = 2
	ServerContext.BEP151Height = 3
	ServerContext.BEP153Height = 4
	ServerContext.BEP159Height = 6
	ServerContext.BEP159Phase2Height = 159
	ServerContext.Config.StateSyncReactor = false
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(context.Bech32PrefixAccAddr, context.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(context.Bech32PrefixValAddr, context.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(context.Bech32PrefixConsAddr, context.Bech32PrefixConsPub)
	config.Seal()
	// create app
	app := NewBinanceChain(logger, memDB, io.Discard)
	logger.Info("BEP159Height", "BEP159Height", ServerContext.BEP159Height)
	logger.Info("BEP159Phase2Height", "BEP159Phase2Height", ServerContext.BEP159Phase2Height)
	logger.Info("BreatheBlockInterval", "BreatheBlockInterval", ServerContext.BreatheBlockInterval)
	logger.Info("IbcChainId", "IbcChainId", ServerContext.IbcChainId)
	logger.Info("BscChainId", "BscChainId", ServerContext.BscChainId)
	logger.Info("BscIbcChainId", "BscIbcChainId", ServerContext.BscIbcChainId)

	// read genesis
	genesisJsonFile, err := os.Open("../asset/mainnet/genesis.json")
	if err != nil {
		panic(err)
	}
	defer genesisJsonFile.Close()
	genesisByteValue, err := ioutil.ReadAll(genesisJsonFile)
	if err != nil {
		panic(err)
	}
	stateBytes := gjson.Get(string(genesisByteValue), "app_state").String()
	n := 100
	accounts := GenAccounts(n)
	app.SetCheckState(abci.Header{})
	app.InitChain(abci.RequestInitChain{
		ChainId:       "Binance-Chain-Tigris",
		Validators:    []abci.ValidatorUpdate{},
		AppStateBytes: []byte(stateBytes),
	})
	ctx := app.BaseApp.DeliverState.Ctx
	for i := 0; i < n; i++ {
		appAcc := accounts[i].AppAccount
		if app.AccountKeeper.GetAccount(ctx, accounts[i].Address) == nil {
			appAcc.BaseAccount.AccountNumber = app.AccountKeeper.GetNextAccountNumber(ctx)
		}
		app.AccountKeeper.SetAccount(ctx, &appAcc)
	}
	app.Commit()
	return app, ctx, accounts
}

func GenSimTxs(app *BinanceChain, msgs []sdk.Msg, expSimPass bool, privs ...crypto.PrivKey,
) (txs []auth.StdTx) {
	accSeqMap := make(map[string][2]int64)
	ctx := app.CheckState.Ctx
	for i, priv := range privs {
		addr := sdk.AccAddress(priv.PubKey().Address())
		logger.Info("addr", "addr", addr)
		accNumSeq, found := accSeqMap[addr.String()]
		if !found {
			acc := app.AccountKeeper.GetAccount(ctx, addr)
			logger.Info("acc", "coins", acc.GetCoins())
			if acc == nil {
				panic(fmt.Sprintf("account %s not found", addr))
			}
			accNumSeq[0] = acc.GetAccountNumber()
			accNumSeq[1] = acc.GetSequence()
		}
		tx := mock.GenTx(msgs[i:i+1], []int64{accNumSeq[0]}, []int64{accNumSeq[1]}, priv)
		res := app.Simulate(nil, tx)
		if expSimPass && !res.IsOK() {
			panic(fmt.Sprintf("simulate failed: %+v", res))
		}
		accSeqMap[addr.String()] = [2]int64{accNumSeq[0], accNumSeq[1] + 1}
		txs = append(txs, tx)
	}
	return txs
}

func ApplyBlock(t *testing.T, app *BinanceChain, ctx sdk.Context, txs []auth.StdTx) (newCtx sdk.Context) {
	height := ctx.BlockHeader().Height + 1
	logger.Debug("ApplyBlock", "height", height)
	header := abci.Header{Height: height}
	validators := app.stakeKeeper.GetSortedBondedValidators(ctx)
	//logger.Debug("ApplyBlock", "validators", validators)
	header.ProposerAddress = validators[0].ConsAddress()
	lastCommitInfo := abci.LastCommitInfo{
		Round: 0,
		Votes: []abci.VoteInfo{},
	}
	for _, validator := range validators {
		lastCommitInfo.Votes = append(lastCommitInfo.Votes, abci.VoteInfo{
			Validator: abci.Validator{
				Address: validator.ConsAddress(),
				Power:   validator.Tokens.RawInt(),
			},
			SignedLastBlock: true,
		})
	}
	app.BeginBlock(abci.RequestBeginBlock{Header: header, LastCommitInfo: lastCommitInfo})
	for _, tx := range txs {
		bz := app.Codec.MustMarshalBinaryLengthPrefixed(tx)
		res := app.DeliverTx(abci.RequestDeliverTx{Tx: bz})
		require.Equal(t, uint32(0), res.Code, res.Log)
	}
	app.EndBlock(abci.RequestEndBlock{Height: height})
	app.Commit()
	newCtx = app.BaseApp.NewContext(sdk.RunTxModeCheck, header)
	return
}

func ApplyEmptyBlocks(t *testing.T, app *BinanceChain, ctx sdk.Context, blockNum int) (newCtx sdk.Context) {
	currentCtx := ctx
	for i := 0; i < blockNum; i++ {
		currentCtx = ApplyBlock(t, app, currentCtx, []auth.StdTx{})
	}
	return currentCtx
}

func ApplyToBreathBlocks(t *testing.T, app *BinanceChain, ctx sdk.Context, breathBlockNum int) (newCtx sdk.Context) {
	currentHeight := ctx.BlockHeader().Height
	blockNum := BREATHE_BLOCK_INTERVAL*breathBlockNum - int(currentHeight%int64(BREATHE_BLOCK_INTERVAL))
	return ApplyEmptyBlocks(t, app, ctx, blockNum)
}

func TestBEP159Distribution(t *testing.T) {
	app, ctx, accs := setupTestForBEP159Test()
	// check genesis validators
	validators := app.stakeKeeper.GetAllValidators(ctx)
	//logger.Info("validators", "validators", validators)
	require.Equal(t, 11, len(validators))
	require.True(t, len(validators[0].DistributionAddr) == 0, "distribution address should be empty")
	// active BEP159
	ctx = ApplyEmptyBlocks(t, app, ctx, 6)
	validators = app.stakeKeeper.GetAllValidators(ctx)
	require.Equal(t, 11, len(validators))
	// logger.Info("validators", "validators", validators)
	// migrate validator to add distribution address
	require.True(t, len(validators[0].DistributionAddr) != 0, "distribution address should not be empty")
	require.Lenf(t, validators[0].StakeSnapshots, 0, "no snapshot yet")
	require.True(t, validators[0].AccumulatedStake.IsZero(), "no AccumulatedStake yet")
	snapshotVals, h, found := app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 1)
	logger.Debug("GetHeightValidatorsByIndex", "snapshotVals", snapshotVals, "h", h, "found", found)
	require.False(t, found, "no validators snapshot yet")
	// no fee got at the beginning of BEP159 activation
	require.True(t, app.CoinKeeper.GetCoins(ctx, validators[0].DistributionAddr).IsZero())
	require.True(t, app.CoinKeeper.GetCoins(ctx, stake.FeeForAllAccAddr).IsZero())
	// transfer tx to make some fees
	transferCoin := sdk.NewCoin("BNB", 100000000)
	inputs := bank.NewInput(accs[0].Address, sdk.Coins{transferCoin})
	outputs := bank.NewOutput(accs[1].Address, sdk.Coins{transferCoin})
	transferMsg := bank.NewMsgSend([]bank.Input{inputs}, []bank.Output{outputs})
	txs := GenSimTxs(app, []sdk.Msg{transferMsg}, true, accs[0].Priv)
	ctx = ApplyBlock(t, app, ctx, txs)
	// pass breath block
	ctx = ApplyToBreathBlocks(t, app, ctx, 1)
	validators = app.stakeKeeper.GetAllValidators(ctx)
	require.Lenf(t, validators[0].StakeSnapshots, 1, "1 snapshot from one breath block")
	require.False(t, validators[0].AccumulatedStake.IsZero(), "had AccumulatedStake")
	snapshotVals, h, found = app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 1)
	logger.Debug("GetHeightValidatorsByIndex", "snapshotVals", snapshotVals, "h", h, "found", found)
	require.True(t, found, "get snapshot in the first breath block after active BEP159")
	snapshotVals, h, found = app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 2)
	require.False(t, found, "only one snapshot")
	// pass 28 breath blocks
	ctx = ApplyToBreathBlocks(t, app, ctx, 28)
	snapshotVals, h, found = app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 1)
	logger.Debug("GetHeightValidatorsByIndex", "snapshotVals", snapshotVals, "h", h, "found", found)
	require.True(t, found)
	require.Len(t, snapshotVals[0].StakeSnapshots, 29)
	// try to create validator
	bondCoin := sdk.NewCoin("BNB", sdk.NewDecWithoutFra(10000*16).RawInt())
	commissionMsg := stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())
	description := stake.NewDescription("validator0", "", "", "")
	createValidatorMsg := stake.MsgCreateValidatorOpen{
		Description:   description,
		Commission:    commissionMsg,
		DelegatorAddr: accs[0].Address,
		ValidatorAddr: sdk.ValAddress(accs[0].Address),
		PubKey:        sdk.MustBech32ifyConsPub(accs[0].Priv.PubKey()),
		Delegation:    bondCoin,
	}
	require.Panics(t, func() {
		txs = GenSimTxs(app, []sdk.Msg{createValidatorMsg}, true, accs[0].Priv)
	})
	// pass one more breath block, activate BEP159Phase2
	ctx = ApplyToBreathBlocks(t, app, ctx, 2)
	snapshotVals, h, found = app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 1)
	require.True(t, found)
	require.Len(t, snapshotVals[0].StakeSnapshots, 30)
	require.True(t, app.CoinKeeper.GetCoins(ctx, validators[0].DistributionAddr).IsZero())
	require.Equal(t, app.CoinKeeper.GetCoins(ctx, stake.FeeForAllAccAddr), sdk.Coins{sdk.NewCoin("BNB", 8)})
	// create new validators, stake number is 16 times of original validators
	createValidatorMsg = stake.MsgCreateValidatorOpen{
		Description:   description,
		Commission:    commissionMsg,
		DelegatorAddr: accs[0].Address,
		ValidatorAddr: sdk.ValAddress(accs[0].Address),
		PubKey:        sdk.MustBech32ifyConsPub(accs[0].Priv.PubKey()),
		Delegation:    bondCoin,
	}
	txs = GenSimTxs(app, []sdk.Msg{createValidatorMsg}, true, accs[0].Priv)
	ctx = ApplyBlock(t, app, ctx, txs)
	require.Equal(t, int64(12), app.stakeKeeper.GetAllValidatorsCount(ctx))
	validators = app.stakeKeeper.GetSortedBondedValidators(ctx)
	// check fees
	ctx = ApplyBlock(t, app, ctx, []auth.StdTx{})
	logger.Debug("feeAddrs", "validator0", validators[0].DistributionAddr, "feeForAll", stake.FeeForAllAccAddr)
	validator0Balance := app.CoinKeeper.GetCoins(ctx, validators[0].DistributionAddr)
	feeForAllBalance := app.CoinKeeper.GetCoins(ctx, stake.FeeForAllAccAddr)
	logger.Debug("feeBalances", "validator0", validator0Balance, "feeForAll", feeForAllBalance)
	require.Equal(t, sdk.Coins{sdk.NewCoin("BNB", 50000000)}, app.CoinKeeper.GetCoins(ctx, validators[0].DistributionAddr))
	require.Equal(t, sdk.Coins{sdk.NewCoin("BNB", 950000008)}, app.CoinKeeper.GetCoins(ctx, stake.FeeForAllAccAddr))

	// check validator just staked
	newValidator, found := app.stakeKeeper.GetValidator(ctx, sdk.ValAddress(accs[0].Address))
	require.True(t, found)
	logger.Info("newValidator", "newValidator", newValidator)
	require.True(t, len(newValidator.DistributionAddr) != 0, "newValidator distribution address should not be empty")
	require.False(t, newValidator.IsBonded(), "newValidator should not be bonded")
	require.Lenf(t, newValidator.StakeSnapshots, 0, "no snapshot yet")
	require.True(t, newValidator.AccumulatedStake.IsZero(), "no AccumulatedStake yet")
	snapshotVals, h, found = app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 1)
	logger.Debug("GetHeightValidatorsByIndex", "snapshotVals", snapshotVals, "h", h, "found", found)
	require.True(t, found)
	require.Equal(t, int64(160), h)

	// apply to next breath block, validator0 accumulated stake not enough, not bounded
	ctx = ApplyToBreathBlocks(t, app, ctx, 1)
	snapshotVals, h, found = app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 1)
	logger.Debug("GetHeightValidatorsByIndex", "snapshotVals", snapshotVals, "h", h, "found", found)
	require.True(t, found, "found validators snapshot")
	require.Len(t, snapshotVals, 11)
	require.Equal(t, int64(165), h)
	require.NotEqual(t, snapshotVals[0].OperatorAddr, accs[0].ValAddress)

	// check fees after distribution
	feeForAllBalance = app.CoinKeeper.GetCoins(ctx, stake.FeeForAllAccAddr)
	logger.Debug("feeBalances", "validator0", validator0Balance, "feeForAll", feeForAllBalance)
	require.Equal(t, app.CoinKeeper.GetCoins(ctx, stake.FeeForAllAccAddr), sdk.Coins{sdk.NewCoin("BNB", 1)})

	// iter all validators, check their fees
	distributionAddrBalanceSum := feeForAllBalance
	var expectedBalance sdk.Coin
	for i, validator := range snapshotVals {
		distributionAddrBalance := app.CoinKeeper.GetCoins(ctx, validator.DistributionAddr)
		feeAddrBalance := app.CoinKeeper.GetCoins(ctx, validator.FeeAddr)
		logger.Debug("distributionAddrBalance", "name", validator.Description.Moniker, "distributionAddrBalance", distributionAddrBalance, "feeAddrBalance", feeAddrBalance)
		require.Lenf(t, distributionAddrBalance, 1, "distributionAddrBalance should have 1 coin")
		distributionAddrBalanceSum = distributionAddrBalanceSum.Plus(distributionAddrBalance)
		if i == 0 {
			expectedBalance = distributionAddrBalance[0].Minus(sdk.NewCoin("BNB", 50000000))
		} else {
			require.False(t, feeAddrBalance.IsZero(), "feeAddrBalance should be zero")
			require.Equal(t, expectedBalance, distributionAddrBalance[0])
		}
	}
	logger.Debug("distributionAddrBalanceSum", "distributionAddrBalanceSum", distributionAddrBalanceSum)
	require.Equal(t, sdk.NewCoin("BNB", 1000000008), distributionAddrBalanceSum[0])

	// apply to next breath block, validator0 become bonded, the first one
	ctx = ApplyToBreathBlocks(t, app, ctx, 1)
	snapshotVals, h, found = app.stakeKeeper.GetHeightValidatorsByIndex(ctx, 1)
	require.Equal(t, int64(170), h)
	require.Equal(t, snapshotVals[0].OperatorAddr, accs[0].ValAddress)
	for _, validator := range snapshotVals {
		distributionAddrBalance := app.CoinKeeper.GetCoins(ctx, validator.DistributionAddr)
		feeAddrBalance := app.CoinKeeper.GetCoins(ctx, validator.FeeAddr)
		logger.Debug("distributionAddrBalance", "name", validator.Description.Moniker, "distributionAddrBalance", distributionAddrBalance, "feeAddrBalance", feeAddrBalance)
		require.True(t, distributionAddrBalance.IsZero())
		require.False(t, feeAddrBalance.IsZero(), "feeAddrBalance should be zero")
	}

	// open this when delegate opens
	//// two more breath block, one for delegator to get into snapshot, one to get rewards
	//// validator1 delegate to validator0
	//delegateMsg := stake.NewMsgDelegate(accs[1].Address, sdk.ValAddress(accs[0].Address), sdk.NewCoin("BNB", sdk.NewDecWithoutFra(10000*16).RawInt()))
	//txs = GenSimTxs(app, []sdk.Msg{delegateMsg}, true, accs[1].Priv)
	//ctx = ApplyBlock(t, app, ctx, txs)
	//delegatorBalance := app.CoinKeeper.GetCoins(ctx, accs[1].Address)
	//logger.Debug("delegatorBalance", "delegatorBalance", delegatorBalance)
	//
	//// need 1 breath block to get into snapshot
	//ctx = ApplyToBreathBlocks(t, app, ctx, 1)
	//require.Equal(t, delegatorBalance, app.CoinKeeper.GetCoins(ctx, accs[1].Address))
	//delegatorBalance = app.CoinKeeper.GetCoins(ctx, accs[1].Address)
	//logger.Debug("delegatorBalance", "delegatorBalance", delegatorBalance)
	//// validator2 delegate to validator0
	//delegateMsg = stake.NewMsgDelegate(accs[2].Address, sdk.ValAddress(accs[0].Address), sdk.NewCoin("BNB", sdk.NewDecWithoutFra(10000*16).RawInt()))
	//txs = GenSimTxs(app, []sdk.Msg{delegateMsg}, true, accs[2].Priv)
	//ctx = ApplyBlock(t, app, ctx, txs)
	//
	//// need 1 breath block to get into distribution addr
	//ctx = ApplyToBreathBlocks(t, app, ctx, 1)
	//require.Equal(t, delegatorBalance, app.CoinKeeper.GetCoins(ctx, accs[1].Address))
	//delegatorBalance = app.CoinKeeper.GetCoins(ctx, accs[1].Address)
	//logger.Debug("delegatorBalance", "delegatorBalance", delegatorBalance)
	//
	//// need 1 more block to distribute rewards
	//ctx = ApplyEmptyBlocks(t, app, ctx, 1)
	//require.NotEqual(t, delegatorBalance, app.CoinKeeper.GetCoins(ctx, accs[1].Address))
	//delegatorBalance = app.CoinKeeper.GetCoins(ctx, accs[1].Address)
	//logger.Debug("delegatorBalance", "delegatorBalance", delegatorBalance)
}
