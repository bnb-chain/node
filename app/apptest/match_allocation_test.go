package apptest

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/binance-chain/node/app"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/param"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"
)

var testFeeConfig order.FeeConfig

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("bnb", "bnbp")
	config.SetBech32PrefixForValidator("bva", "bvap")
	config.SetBech32PrefixForConsensusNode("bca", "bcap")
	config.Seal()
	initFeeConfig()
}

func initFeeConfig() {
	testFeeConfig = order.NewFeeConfig()
	// 500/1000000 = 0.0005
	testFeeConfig.FeeRateNative = 500
	// 1000/1000000 = 0.001
	testFeeConfig.FeeRate = 1000
	// 20000/100000000 = 0.0002
	testFeeConfig.ExpireFeeNative = 2e4
	testFeeConfig.ExpireFee = 1e5
	testFeeConfig.IOCExpireFeeNative = 1e4
	testFeeConfig.IOCExpireFee = 5e4
	testFeeConfig.CancelFeeNative = 2e4
	testFeeConfig.CancelFee = 1e5

	testFeeConfig.MakerFeeRateNative = 100
	testFeeConfig.MakerFeeRate = 200
	testFeeConfig.TakerFeeRateNative = 500
	testFeeConfig.TakerFeeRate = 1000
}

func SetupTest(initPrices ...int64) (crypto.Address, sdk.Context, []sdk.Account) {
	// for old match engine
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP19, math.MaxInt64)
	addr := secp256k1.GenPrivKey().PubKey().Address()
	accAddr := sdk.AccAddress(addr)
	baseAcc := auth.BaseAccount{Address: accAddr}
	genTokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,accAddr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil), 0}
	genAccs := make([]app.GenesisAccount, 1)
	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genAccs[0] = app.NewGenesisAccount(appAcc, valAddr)
	genesisState := app.GenesisState{
		Tokens:       genTokens,
		Accounts:     genAccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}
	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}
	testApp.InitChain(abci.RequestInitChain{
		Validators: []abci.ValidatorUpdate{},
		AppStateBytes: stateBytes})
	// it is required in fee distribution during end block
	testApp.ValAddrCache.SetAccAddr(sdk.ConsAddress(valAddr), appAcc.Address)
	ctx := testApp.DeliverState.Ctx
	prices := []int64{int64(10e8), int64(10e8), int64(10e8)}
	// update 3 pairs' default prices
	for i, price := range initPrices {
		prices[i] = price
	}
	bnbbtcPair := types.NewTradingPair("BTC-000", "BNB", prices[0])
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, bnbbtcPair)
	testApp.DexKeeper.AddEngine(bnbbtcPair)
	bnbethPair := types.NewTradingPair("ETH-000", "BNB", prices[1])
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, bnbethPair)
	testApp.DexKeeper.AddEngine(bnbethPair)
	ethbtcPair := types.NewTradingPair("BTC-000", "ETH-000", prices[2])
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, ethbtcPair)
	testApp.DexKeeper.AddEngine(ethbtcPair)
	for _, v := range testApp.DexKeeper.PairMapper.ListAllTradingPairs(ctx) {
		msg := fmt.Sprintf("%s_%s: %d|%d|%d", v.BaseAssetSymbol, v.QuoteAssetSymbol, v.ListPrice, v.LotSize, v.TickSize)
		testApp.Logger.Info(msg)
	}
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	testApp.DexKeeper.ClearOrderBook("ETH-000_BNB")
	testApp.DexKeeper.ClearOrderBook("BTC-000_ETH-000")
	testApp.DexKeeper.FeeManager.UpdateConfig(testFeeConfig)
	coins := sdk.Coins{
		sdk.NewCoin("BNB", 100000e8),
		sdk.NewCoin("BTC-000", 100000e8),
		sdk.NewCoin("ETH-000", 100000e8)}
	var accs []sdk.Account
	for i := 0; i < 10; i++ {
		privKey := ed25519.GenPrivKey()
		pubKey := privKey.PubKey()
		addr := sdk.AccAddress(pubKey.Address())
		acc := &auth.BaseAccount{
			Address: addr,
			Coins:   coins,
		}
		appAcc := &common.AppAccount{BaseAccount: *acc}
		if testApp.AccountKeeper.GetAccount(ctx, acc.GetAddress()) == nil {
			appAcc.BaseAccount.AccountNumber = testApp.AccountKeeper.GetNextAccountNumber(ctx)
		}
		testApp.AccountKeeper.SetAccount(ctx, appAcc)
		accs = append(accs, acc)
	}
	return valAddr, ctx, accs
}

func SetupTest_new(initPrices ...int64) (crypto.Address, sdk.Context, []sdk.Account) {
	// for new match engine
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP19, -1)
	addr := secp256k1.GenPrivKey().PubKey().Address()
	accAddr := sdk.AccAddress(addr)
	baseAcc := auth.BaseAccount{Address: accAddr}
	genTokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,accAddr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil), 0}
	genAccs := make([]app.GenesisAccount, 1)
	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genAccs[0] = app.NewGenesisAccount(appAcc, valAddr)
	genesisState := app.GenesisState{
		Tokens:       genTokens,
		Accounts:     genAccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}
	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}
	testApp.InitChain(abci.RequestInitChain{
		Validators: []abci.ValidatorUpdate{},
		AppStateBytes: stateBytes})
	// it is required in fee distribution during end block
	testApp.ValAddrCache.SetAccAddr(sdk.ConsAddress(valAddr), appAcc.Address)
	ctx := testApp.DeliverState.Ctx
	prices := []int64{int64(10e8), int64(10e8), int64(10e8)}
	// update 3 pairs' default prices
	for i, price := range initPrices {
		prices[i] = price
	}
	bnbbtcPair := types.NewTradingPair("BTC-000", "BNB", prices[0])
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, bnbbtcPair)
	testApp.DexKeeper.AddEngine(bnbbtcPair)
	bnbethPair := types.NewTradingPair("ETH-000", "BNB", prices[1])
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, bnbethPair)
	testApp.DexKeeper.AddEngine(bnbethPair)
	ethbtcPair := types.NewTradingPair("BTC-000", "ETH-000", prices[2])
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, ethbtcPair)
	testApp.DexKeeper.AddEngine(ethbtcPair)
	for _, v := range testApp.DexKeeper.PairMapper.ListAllTradingPairs(ctx) {
		msg := fmt.Sprintf("%s_%s: %d|%d|%d", v.BaseAssetSymbol, v.QuoteAssetSymbol, v.ListPrice, v.LotSize, v.TickSize)
		testApp.Logger.Info(msg)
	}
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	testApp.DexKeeper.ClearOrderBook("ETH-000_BNB")
	testApp.DexKeeper.ClearOrderBook("BTC-000_ETH-000")
	testApp.DexKeeper.FeeManager.UpdateConfig(testFeeConfig)
	coins := sdk.Coins{
		sdk.NewCoin("BNB", 100000e8),
		sdk.NewCoin("BTC-000", 100000e8),
		sdk.NewCoin("ETH-000", 100000e8)}
	var accs []sdk.Account
	for i := 0; i < 10; i++ {
		privKey := ed25519.GenPrivKey()
		pubKey := privKey.PubKey()
		addr := sdk.AccAddress(pubKey.Address())
		acc := &auth.BaseAccount{
			Address: addr,
			Coins:   coins,
		}
		appAcc := &common.AppAccount{BaseAccount: *acc}
		if testApp.AccountKeeper.GetAccount(ctx, acc.GetAddress()) == nil {
			appAcc.BaseAccount.AccountNumber = testApp.AccountKeeper.GetNextAccountNumber(ctx)
		}
		testApp.AccountKeeper.SetAccount(ctx, appAcc)
		accs = append(accs, acc)
	}
	return valAddr, ctx, accs
}

// for common block
func UpdateContextC(addr crypto.Address, ctx sdk.Context, height int64) sdk.Context {
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: addr, Height: height}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: addr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx
	return ctx
}

// for breath block
func UpdateContextB(addr crypto.Address, ctx sdk.Context, height int64, tNow time.Time) sdk.Context {
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: addr, Height: height, Time: tNow}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: addr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx
	return ctx
}

func GetOrderId(add sdk.AccAddress, seq int64, ctx sdk.Context) string {
	acc := testApp.AccountKeeper.GetAccount(ctx, add)
	if acc.GetSequence() != seq {
		err := acc.SetSequence(seq)
		if err != nil {
			panic(err)
		}
		testApp.AccountKeeper.SetAccount(ctx, acc)
	}
	oid := fmt.Sprintf("%X-%d", add, seq)
	return oid
}

func GetOrderBook(pair string) ([]level, []level) {
	buys := make([]level, 0)
	sells := make([]level, 0)
	orderbooks := testApp.DexKeeper.GetOrderBookLevels(pair, 25)
	for _, l := range orderbooks {
		if l.BuyPrice != 0 {
			buys = append(buys, level{price: l.BuyPrice, qty: l.BuyQty})
		}
		if l.SellPrice != 0 {
			sells = append(sells, level{price: l.SellPrice, qty: l.SellQty})
		}
	}
	return buys, sells
}

func ResetAccount(ctx sdk.Context, addr sdk.AccAddress, ccy1 int64, ccy2 int64, ccy3 int64) {
	acc := testApp.AccountKeeper.GetAccount(ctx, addr)
	acc.SetCoins(sdk.Coins{
		sdk.NewCoin("BNB", ccy1),
		sdk.NewCoin("BTC-000", ccy2),
		sdk.NewCoin("ETH-000", ccy3),
	})
	testApp.AccountKeeper.SetAccount(ctx, acc)
}

func Test_Match_Allocation(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	//#1 cannot buy with more than they have
	msg1_1 := order.NewNewOrderMsg(addr0, GetOrderId(addr0, 0, ctx), 1, "BTC-000_BNB", 1e8, 75000e8)
	_, err := testClient.DeliverTxSync(msg1_1, testApp.Codec)
	assert.NoError(err)

	msg1_2 := order.NewNewOrderMsg(addr0, GetOrderId(addr0, 1, ctx), 1, "BTC-000_BNB", 1e8, 75000e8)
	res, err := testClient.DeliverTxSync(msg1_2, testApp.Codec)
	assert.NoError(err)

	assert.Equal(true, strings.Contains(res.Log, "do not have enough token to lock"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(25000e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(75000e8), GetLocked(ctx, addr0, "BNB"))

	//#2 cannot sell more than they have
	msg2_1 := order.NewNewOrderMsg(addr1, GetOrderId(addr1, 0, ctx), 2, "BTC-000_BNB", 1e8, 60000e8)
	_, err = testClient.DeliverTxSync(msg2_1, testApp.Codec)
	assert.NoError(err)

	msg2_2 := order.NewNewOrderMsg(addr1, GetOrderId(addr1, 1, ctx), 2, "BTC-000_BNB", 1e8, 60000e8)
	res, err = testClient.DeliverTxSync(msg2_2, testApp.Codec)
	assert.NoError(err)

	assert.Equal(true, strings.Contains(res.Log, "do not have enough token to lock"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(40000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(60000e8), GetLocked(ctx, addr1, "BTC-000"))

	//#3 cancel will return fund
	msg3_1 := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msg1_1.Id)
	_, err = testClient.DeliverTxSync(msg3_1, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr0, "BNB"))

	msg3_2 := order.NewNewOrderMsg(addr0, GetOrderId(addr0, 2, ctx), 1, "BTC-000_BNB", 1e8, 75000e8)
	_, err = testClient.DeliverTxSync(msg3_2, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(24999.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(75000e8), GetLocked(ctx, addr0, "BNB"))

	msg3_3 := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msg3_2.Id)
	_, err = testClient.DeliverTxSync(msg3_3, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9996e8), GetAvail(ctx, addr0, "BNB"))

	msg3_4 := order.NewCancelOrderMsg(addr1, "BTC-000_BNB", msg2_1.Id)
	_, err = testClient.DeliverTxSync(msg3_4, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr1, "BNB"))

	//#4, test different price and allocation rules

	/*
	   [- min surplus (absolute leftover volume)]
	   sum    sell    price    buy    sum    exec    imbal
	   150            12       30     30     30      -120
	   150            11              30     30      -120
	   150            10       10     40     40      -110
	   150            9        20     60     60      -90
	   150    25      8        30     90     90      -60
	   125    25      7               90     90      -35
	   100    100     6               90     90      -10
	*/

	msg := order.NewNewOrderMsg(addr2, GetOrderId(addr2, 0, ctx), 1, "BTC-000_BNB", 12e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 1, ctx), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 2, ctx), 1, "BTC-000_BNB", 9e8, 20e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 3, ctx), 1, "BTC-000_BNB", 8e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 0, ctx), 2, "BTC-000_BNB", 8e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 1, ctx), 2, "BTC-000_BNB", 7e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 2, ctx), 2, "BTC-000_BNB", 6e8, 100e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(6e8), lastPx)
	assert.Equal(4, len(trades))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99459.73e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100539.73e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 15 @ 12
	   sell 10 @ 6
	   [- min surplus (absolute leftover volume)
	    - orders from earlier are filled first]
	   sum    sell    price    buy    sum    exec    imbal
	   70             12       15     15     15      -55
	   70             11              15     15      -55
	   70             10              15     15      -55
	   70             9               15     15      -55
	   70     25      8               15     15      -55
	   45     25      7               15     15      -30
	   20     20      6               15     15      -5
	*/
	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 4, ctx), 1, "BTC-000_BNB", 12e8, 15e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr1, GetOrderId(addr1, 1, ctx), 2, "BTC-000_BNB", 6e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(6e8), lastPx)
	assert.Equal(2, len(trades))

	assert.Equal(int64(100105e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99369.6850e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100599.7e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100029.9848e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 4)

	/*
	   incoming
	   buy 5 @ 12
	   buy 5 @ 10
	   buy 10 @ 8
	   buy 5 @ 7
	   [- max matched volume]
	   sum    sell    price    buy    sum    exec    imbal
	   55             12       5      5      5       -50
	   55             11              5      5       -50
	   55             10       5      10     10      -45
	   55             9               10     10      -45
	   55     25      8        10     20     20      -35
	   30     25      7        5      25     25      -5
	   5      5       6               25     5       20
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 5, ctx), 1, "BTC-000_BNB", 12e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 6, ctx), 1, "BTC-000_BNB", 10e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 7, ctx), 1, "BTC-000_BNB", 8e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 8, ctx), 1, "BTC-000_BNB", 7e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 5)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(4, len(trades))

	assert.Equal(int64(100130e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99194.5975e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100739.63e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100064.9673e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))

	/*
	   incoming
	   buy 10 @ 13
	   buy 10 @ 10
	   sell 25 @ 8
	   [- adjust market pressure, sell side, ap > rp - 5% => lowest]
	   sum    sell    price    buy    sum    exec    imbal
	   50             13       10     10     10      -40
	   50             12              10     10      -40
	   50             11              10     10      -40
	   50             10       10     20     20      -30
	   50             9               20     20      -30
	   50     50      8               20     20      -30
	   5      5       7               20     5       15
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 9, ctx), 1, "BTC-000_BNB", 13e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 10, ctx), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 3, ctx), 2, "BTC-000_BNB", 8e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 6)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100150e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99034.5175e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99825e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100899.55e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 13
	   buy 10 @ 10
	   sell 10 @ 11
	   [- adjust market pressure, sell side, ap > rp - 5% => lowest]
	   sum    sell    price    buy    sum    exec    imbal
	   45             13       10     10     10      -35
	   45             12              10     10      -35
	   45     10      11              10     10      -35
	   35             10       10     20     10      -15
	   35             9               20     10      -15
	   35     35      8               20     10      -15
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 11, ctx), 1, "BTC-000_BNB", 13e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 4, ctx), 2, "BTC-000_BNB", 11e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 12, ctx), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 7)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(2, len(trades))

	assert.Equal(int64(100170e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(98874.4375e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99815e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(101059.47e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 12
	   buy 10 @ 9
	   buy 10 @ 9
	   [- adjust market pressure, buy side, else: min(rp + 5%, highest)
	    - orders with same price and block height are split proportionally]
	   sum    sell    price    buy    sum    exec    imbal
	   25             12       10     10     10      -15
	   25     10      11              10     10      -15
	   15             10              10     10      -5
	   15             9        20     20     15      5
	   15     15      8               20     15      5
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 13, ctx), 1, "BTC-000_BNB", 12e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 14, ctx), 1, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 15, ctx), 1, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 8)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8.4e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100185e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(98613.3745e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99815e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(101185.4070e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 50 @ 20
	   sell 10 @ 10
	   sell 10 @ 9
	   sell 10 @ 7
	   [- adjust market pressure, buy side, ap > rp + 5% => lowest]
	   big order (up)
	   sum    sell    price    buy    sum    exec    imbal
	   40             20       50     50     40      10
	   40             12              50     40      10
	   40     10      11              50     40      10
	   30     10      10              50     30      20
	   20     10      9        15     65     20      45
	   10             8               65     10      55
	   10     10      7               65     10      55
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 16, ctx), 1, "BTC-000_BNB", 20e8, 50e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 5, ctx), 2, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 6, ctx), 2, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 7, ctx), 2, "BTC-000_BNB", 7e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(4, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 9)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(4, len(trades))

	assert.Equal(int64(100225e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(97973.1545e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(335e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99785e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(101625.1870e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 30 @ 15
	   sell 10 @ 17
	   sell 10 @ 15
	   sell 30 @ 9
	   [- min surplus (absolute leftover volume)]
	   big order (up)
	   sum    sell    price    buy    sum    exec    imbal
	   50             20       10     10     10      -40
	   50     10      17              10     10      -40
	   40     10      15       30     40     30      0
	   30             12              40     30      10
	   30             11              40     30      10
	   30             10              40     30      10
	   30     30      9        15     55     15      25
	*/

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 8, ctx), 2, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 17, ctx), 1, "BTC-000_BNB", 15e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 9, ctx), 2, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 10, ctx), 2, "BTC-000_BNB", 9e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 10)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(15e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100265e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(97572.8545e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99735e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(102224.8870e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 17
	   buy 10 @ 15
	   buy 10 @ 11
	   sell 10 @ 16
	   sell 10 @ 15
	   sell 30 @ 9
	   [- rp or close to rp]
	   big order (down)
	   sum    sell    price    buy    sum    exec    imbal
	   60     10      17       10     10     10      -50
	   50     10      16              10     10      -40
	   40     10      15       10     20     20      -20
	   30             12              20     20      -10
	   30             11       10     30     30      0
	   30             10              30     30      0
	   30     30      9        15     45     30      15
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 18, ctx), 1, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 11, ctx), 2, "BTC-000_BNB", 16e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 19, ctx), 1, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 12, ctx), 2, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 20, ctx), 1, "BTC-000_BNB", 11e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 13, ctx), 2, "BTC-000_BNB", 9e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(4, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 11)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100295e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(97242.6895e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99685e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(102554.7220e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 30 @ 17
	   [- max matched volume]
	   big order (up)
	   sum    sell    price    buy    sum    exec    imbal
	   30     10      17       30     30     30      0
	   20     10      16              30     20      10
	   10     10      15              30     10      20
	   0              12              30     0       30
	   0              11              30     0       30
	   0              10              30     0       30
	   0              9        15     45     0       45
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 21, ctx), 1, "BTC-000_BNB", 17e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 12)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(17e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100325e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(96732.4345e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99685e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(103064.4670e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 17
	   buy 10 @ 12
	   sell 30 @ 10
	   [ - adjust market pressure, sell side, ap < rp - 5% => highest]
	   big order (down)
	   sum    sell    price    buy    sum    exec    imbal
	   30             17       10     10     10      -20
	   30             16              10     10      -20
	   30             15              10     10      -20
	   30             12       10     20     20      -10
	   30             11              20     20      -10
	   30     30      10              20     20      -10
	   0              9        15     35     0       35
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 22, ctx), 1, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 23, ctx), 1, "BTC-000_BNB", 12e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 14, ctx), 2, "BTC-000_BNB", 10e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 13)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(12e8), lastPx)
	assert.Equal(2, len(trades))

	assert.Equal(int64(100345e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(96492.3145e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99655e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(103304.3470e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
}

func Test_Match_Allocation_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	//#1 cannot buy with more than they have
	msg1_1 := order.NewNewOrderMsg(addr0, GetOrderId(addr0, 0, ctx), 1, "BTC-000_BNB", 1e8, 75000e8)
	_, err := testClient.DeliverTxSync(msg1_1, testApp.Codec)
	assert.NoError(err)

	msg1_2 := order.NewNewOrderMsg(addr0, GetOrderId(addr0, 1, ctx), 1, "BTC-000_BNB", 1e8, 75000e8)
	res, err := testClient.DeliverTxSync(msg1_2, testApp.Codec)
	assert.NoError(err)

	assert.Equal(true, strings.Contains(res.Log, "do not have enough token to lock"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(25000e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(75000e8), GetLocked(ctx, addr0, "BNB"))

	//#2 cannot sell more than they have
	msg2_1 := order.NewNewOrderMsg(addr1, GetOrderId(addr1, 0, ctx), 2, "BTC-000_BNB", 1e8, 60000e8)
	_, err = testClient.DeliverTxSync(msg2_1, testApp.Codec)
	assert.NoError(err)

	msg2_2 := order.NewNewOrderMsg(addr1, GetOrderId(addr1, 1, ctx), 2, "BTC-000_BNB", 1e8, 60000e8)
	res, err = testClient.DeliverTxSync(msg2_2, testApp.Codec)
	assert.NoError(err)

	assert.Equal(true, strings.Contains(res.Log, "do not have enough token to lock"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(40000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(60000e8), GetLocked(ctx, addr1, "BTC-000"))

	//#3 cancel will return fund
	msg3_1 := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msg1_1.Id)
	_, err = testClient.DeliverTxSync(msg3_1, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr0, "BNB"))

	msg3_2 := order.NewNewOrderMsg(addr0, GetOrderId(addr0, 2, ctx), 1, "BTC-000_BNB", 1e8, 75000e8)
	_, err = testClient.DeliverTxSync(msg3_2, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(24999.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(75000e8), GetLocked(ctx, addr0, "BNB"))

	msg3_3 := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msg3_2.Id)
	_, err = testClient.DeliverTxSync(msg3_3, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9996e8), GetAvail(ctx, addr0, "BNB"))

	msg3_4 := order.NewCancelOrderMsg(addr1, "BTC-000_BNB", msg2_1.Id)
	_, err = testClient.DeliverTxSync(msg3_4, testApp.Codec)
	assert.NoError(err)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr1, "BNB"))

	//#4, test different price and allocation rules

	/*
	   [- min surplus (absolute leftover volume)]
	   sum    sell    price    buy    sum    exec    imbal
	   150            12       30     30     30      -120
	   150            11              30     30      -120
	   150            10       10     40     40      -110
	   150            9        20     60     60      -90
	   150    25      8        30     90     90      -60
	   125    25      7               90     90      -35
	   100    100     6               90     90      -10
	*/

	msg := order.NewNewOrderMsg(addr2, GetOrderId(addr2, 0, ctx), 1, "BTC-000_BNB", 12e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 1, ctx), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 2, ctx), 1, "BTC-000_BNB", 9e8, 20e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 3, ctx), 1, "BTC-000_BNB", 8e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 0, ctx), 2, "BTC-000_BNB", 8e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 1, ctx), 2, "BTC-000_BNB", 7e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 2, ctx), 2, "BTC-000_BNB", 6e8, 100e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(6e8), lastPx)
	assert.Equal(4, len(trades))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99459.73e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100539.73e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 15 @ 12
	   sell 10 @ 6
	   [- min surplus (absolute leftover volume)
	    - orders from earlier are filled first]
	   sum    sell       price    buy    sum    exec    imbal
	   70                12       15     15     15      -55
	   70                11              15     15      -55
	   70                10              15     15      -55
	   70                9               15     15      -55
	   70     25(m)      8               15     15      -55
	   45     25(m)      7               15     15      -30
	   20     20(10m,10) 6               15     15      -5
	*/
	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 4, ctx), 1, "BTC-000_BNB", 12e8, 15e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr1, GetOrderId(addr1, 1, ctx), 2, "BTC-000_BNB", 6e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(6e8), lastPx)
	assert.Equal(2, len(trades))

	assert.Equal(int64(100105e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99369.6850e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100599.7e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100029.9848e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 4)

	/*
	   incoming
	   buy 5 @ 12
	   buy 5 @ 10
	   buy 10 @ 8
	   buy 5 @ 7
	   [- max matched volume]
	   sum    sell    price    buy    sum    exec    imbal
	   55             12       5      5      5       -50
	   55             11              5      5       -50
	   55             10       5      10     10      -45
	   55             9               10     10      -45
	   55     25(m)   8        10     20     20      -35
	   30     25(m)   7        5      25     25      -5
	   5      5(m)    6               25     5       20
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 5, ctx), 1, "BTC-000_BNB", 12e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 6, ctx), 1, "BTC-000_BNB", 10e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 7, ctx), 1, "BTC-000_BNB", 8e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 8, ctx), 1, "BTC-000_BNB", 7e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 5)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(8, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	//5*5/25=1
	//5*5/25=1
	//5*10/25=2
	//5*5/25=1
	assert.Equal(int64(6e8), trades[0].LastPx)
	assert.Equal(int64(1e8), trades[0].LastQty)
	assert.Equal(int64(6e8), trades[1].LastPx)
	assert.Equal(int64(1e8), trades[1].LastQty)
	assert.Equal(int64(6e8), trades[2].LastPx)
	assert.Equal(int64(2e8), trades[2].LastQty)
	assert.Equal(int64(6e8), trades[3].LastPx)
	assert.Equal(int64(1e8), trades[3].LastQty)
	//5-1=4
	//5-1=4
	//10-2=8
	//5-1=4
	assert.Equal(int64(7e8), trades[4].LastPx)
	assert.Equal(int64(4e8), trades[4].LastQty)
	assert.Equal(int64(7e8), trades[5].LastPx)
	assert.Equal(int64(4e8), trades[5].LastQty)
	assert.Equal(int64(7e8), trades[6].LastPx)
	assert.Equal(int64(8e8), trades[6].LastQty)
	assert.Equal(int64(7e8), trades[7].LastPx)
	assert.Equal(int64(4e8), trades[7].LastQty)

	assert.Equal(int64(100130e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99199.6000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100739.63e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100059.9698e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))

	/*
	   incoming
	   buy 10 @ 13
	   buy 10 @ 10
	   sell 25 @ 8
	   [- adjust market pressure, sell side, ap > rp - 5% => lowest]
	   sum    sell        price    buy    sum    exec    imbal
	   50                 13       10     10     10      -40
	   50                 12              10     10      -40
	   50                 11              10     10      -40
	   50                 10       10     20     20      -30
	   50                 9               20     20      -30
	   50     50(25m,25m) 8               20     20      -30
	   5      5(m)        7               20     5       15
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 9, ctx), 1, "BTC-000_BNB", 13e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 10, ctx), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 3, ctx), 2, "BTC-000_BNB", 8e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 6)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	//5*10/20=2.5
	//5*10/20=2.5
	assert.Equal(int64(7e8), trades[0].LastPx)
	assert.Equal(int64(2.5e8), trades[0].LastQty)
	assert.Equal(int64(7e8), trades[1].LastPx)
	assert.Equal(int64(2.5e8), trades[1].LastQty)
	//10-2.5=7.5
	//10-2.5=7.5
	assert.Equal(int64(8e8), trades[2].LastPx)
	assert.Equal(int64(7.5e8), trades[2].LastQty)
	assert.Equal(int64(8e8), trades[3].LastPx)
	assert.Equal(int64(7.5e8), trades[3].LastQty)

	assert.Equal(int64(100150e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99044.5225e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99825e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100894.5525e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 13
	   buy 10 @ 10
	   sell 10 @ 11
	   [- adjust market pressure, sell side, ap > rp - 5% => lowest]
	   sum    sell        price    buy    sum    exec    imbal
	   45                 13       10     10     10      -35
	   45                 12              10     10      -35
	   45     10          11              10     10      -35
	   35                 10       10     20     10      -15
	   35                 9               20     10      -15
	   35     35(10m,25m) 8               20     10      -15
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 11, ctx), 1, "BTC-000_BNB", 13e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 4, ctx), 2, "BTC-000_BNB", 11e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 12, ctx), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 7)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(2, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(10e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[0].LastQty)

	assert.Equal(int64(100170e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(98884.4425e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99815e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(101054.4725e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 12
	   buy 10 @ 9
	   buy 10 @ 9
	   [- adjust market pressure, buy side, else: min(rp + 5%, highest)
	    - orders with same price and block height are split proportionally]
	   sum    sell    price    buy    sum    exec    imbal
	   25             12       10     10     10      -15
	   25     10(m)   11              10     10      -15
	   15             10              10     10      -5
	   15             9        20     20     15      5
	   15     15(m)   8               20     15      5
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 13, ctx), 1, "BTC-000_BNB", 12e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 14, ctx), 1, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 15, ctx), 1, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 8)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8.4e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(8e8), trades[0].LastPx)
	assert.Equal(int64(10e8), trades[0].LastQty)
	assert.Equal(int64(8e8), trades[1].LastPx)
	assert.Equal(int64(2.5e8), trades[1].LastQty)
	assert.Equal(int64(8e8), trades[2].LastPx)
	assert.Equal(int64(2.5e8), trades[2].LastQty)

	assert.Equal(int64(100185e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(98629.3825e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99815e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(101174.4125e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 50 @ 20
	   sell 10 @ 10
	   sell 10 @ 9
	   sell 10 @ 7
	   [- adjust market pressure, buy side, ap > rp + 5% => lowest]
	   big order (up)
	   sum    sell    price    buy           sum    exec    imbal
	   40             20       50            50     40      10
	   40             12                     50     40      10
	   40     10(m)   11                     50     40      10
	   30     10      10                     50     30      20
	   20     10      9        15(7.5m,7.5m) 65     20      45
	   10             8                      65     10      55
	   10     10      7                      65     10      55
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 16, ctx), 1, "BTC-000_BNB", 20e8, 50e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 5, ctx), 2, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 6, ctx), 2, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 7, ctx), 2, "BTC-000_BNB", 7e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(4, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 9)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(10e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastQty)
	assert.Equal(int64(10e8), trades[2].LastQty)
	assert.Equal(int64(10e8), trades[3].LastQty)

	assert.Equal(int64(100225e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(97989.1625e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(335e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99785e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(101614.1925e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 30 @ 15
	   sell 10 @ 17
	   sell 10 @ 15
	   sell 30 @ 9
	   [- min surplus (absolute leftover volume)]
	   big order (up)
	   sum    sell    price    buy           sum    exec    imbal
	   50             20       10(m)         10     10      -40
	   50     10      17                     10     10      -40
	   40     10      15       30            40     30      0
	   30             12                     40     30      10
	   30             11                     40     30      10
	   30             10                     40     30      10
	   30     30      9        15(7.5m,7.5m) 55     15      25
	*/

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 8, ctx), 2, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 17, ctx), 1, "BTC-000_BNB", 15e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 9, ctx), 2, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 10, ctx), 2, "BTC-000_BNB", 9e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 10)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(15e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	//10*30/40=7.5
	//10*10/40=2.5
	assert.Equal(int64(20e8), trades[0].LastPx)
	assert.Equal(int64(7.5e8), trades[0].LastQty)
	assert.Equal(int64(20e8), trades[1].LastPx)
	assert.Equal(int64(2.5e8), trades[1].LastQty)
	//30-7.5=22.5
	//10-2.5=7.5
	assert.Equal(int64(15e8), trades[2].LastPx)
	assert.Equal(int64(22.5e8), trades[2].LastQty)
	assert.Equal(int64(15e8), trades[3].LastPx)
	assert.Equal(int64(7.5e8), trades[3].LastQty)

	assert.Equal(int64(100265e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(97538.8375e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99735e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(102263.8675e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 17
	   buy 10 @ 15
	   buy 10 @ 11
	   sell 10 @ 16
	   sell 10 @ 15
	   sell 30 @ 9
	   [- rp or close to rp]
	   big order (down)
	   sum    sell    price    buy           sum    exec    imbal
	   60     10(m)   17       10            10     10      -50
	   50     10      16                     10     10      -40
	   40     10      15       10            20     20      -20
	   30             12                     20     20      -10
	   30             11       10            30     30      0
	   30             10                     30     30      0
	   30     30      9        15(7.5m,7.5m) 45     30      15
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 18, ctx), 1, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 11, ctx), 2, "BTC-000_BNB", 16e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 19, ctx), 1, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 12, ctx), 2, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 20, ctx), 1, "BTC-000_BNB", 11e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 13, ctx), 2, "BTC-000_BNB", 9e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(4, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 11)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(10e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastQty)
	assert.Equal(int64(10e8), trades[2].LastQty)

	assert.Equal(int64(100295e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(97208.6725e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99685e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(102593.7025e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 30 @ 17
	   [- max matched volume]
	   big order (up)
	   sum    sell    price    buy           sum    exec    imbal
	   30     10(m)   17       30            30     30      0
	   20     10(m)   16                     30     20      10
	   10     10(m)   15                     30     10      20
	   0              12                     30     0       30
	   0              11                     30     0       30
	   0              10                     30     0       30
	   0              9        15(7.5m,7.5m) 45     0       45
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 21, ctx), 1, "BTC-000_BNB", 17e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 12)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(17e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(15e8), trades[0].LastPx)
	assert.Equal(int64(10e8), trades[0].LastQty)
	assert.Equal(int64(16e8), trades[1].LastPx)
	assert.Equal(int64(10e8), trades[1].LastQty)
	assert.Equal(int64(17e8), trades[2].LastPx)
	assert.Equal(int64(10e8), trades[2].LastQty)

	assert.Equal(int64(100325e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(96728.4325e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99685e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(103073.4625e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))

	/*
	   incoming
	   buy 10 @ 17
	   buy 10 @ 12
	   sell 30 @ 10
	   [ - adjust market pressure, sell side, ap < rp - 5% => highest]
	   big order (down)
	   sum    sell    price    buy           sum    exec    imbal
	   30             17       10            10     10      -20
	   30             16                     10     10      -20
	   30             15                     10     10      -20
	   30             12       10            20     20      -10
	   30             11                     20     20      -10
	   30     30      10                     20     20      -10
	   0              9        15(7.5m,7.5m) 35     0       35
	*/

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 22, ctx), 1, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr2, GetOrderId(addr2, 23, ctx), 1, "BTC-000_BNB", 12e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = order.NewNewOrderMsg(addr3, GetOrderId(addr3, 14, ctx), 2, "BTC-000_BNB", 10e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 13)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(12e8), lastPx)
	assert.Equal(2, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(10e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastQty)

	assert.Equal(int64(100345e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(96488.3125e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99655e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(103313.3425e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
}