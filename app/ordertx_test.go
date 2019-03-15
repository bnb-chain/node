package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/node/common/fees"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex"
	o "github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/param"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"

	abci "github.com/tendermint/tendermint/abci/types"
	ty "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

type level struct {
	price utils.Fixed8
	qty   utils.Fixed8
}

func getOrderBook(pair string) ([]level, []level) {
	buys := make([]level, 0)
	sells := make([]level, 0)
	orderbooks := testApp.DexKeeper.GetOrderBookLevels(pair, 100)
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

func genOrderID(add sdk.AccAddress, seq int64, ctx sdk.Context, am auth.AccountKeeper) string {
	acc := am.GetAccount(ctx, add)
	if acc.GetSequence() != seq {
		err := acc.SetSequence(seq)
		if err != nil {
			panic(err)
		}
		am.SetAccount(ctx, acc)
	}
	oid := fmt.Sprintf("%X-%d", add, seq)
	return oid
}

func newTestFeeConfig() o.FeeConfig {
	feeConfig := o.NewFeeConfig()
	feeConfig.FeeRateNative = 500
	feeConfig.FeeRate = 1000
	feeConfig.ExpireFeeNative = 2e4
	feeConfig.ExpireFee = 1e5
	feeConfig.IOCExpireFeeNative = 1e4
	feeConfig.IOCExpireFee = 5e4
	feeConfig.CancelFeeNative = 2e4
	feeConfig.CancelFee = 1e5
	return feeConfig
}

func Test_handleNewOrder_CheckTx(t *testing.T) {
	assert := assert.New(t)
	ctx := testApp.NewContext(sdk.RunTxModeCheck, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, types.NewTradingPair("BTC-000", "BNB", 1e8))

	am := testApp.AccountKeeper
	acc := Account(0)
	acc2 := Account(1)
	add := acc.GetAddress()
	add2 := acc2.GetAddress()
	msg := o.NewNewOrderMsg(add, genOrderID(add, 0, ctx, am), 1, "BTC-000_BNB", 355e8, 100e8)
	res, e := testClient.CheckTxSync(msg, testApp.Codec)
	assert.NotEqual(uint32(0), res.Code)
	assert.Nil(e)
	assert.Regexp(".*do not have enough token to lock.*", res.GetLog())
	assert.Equal(int64(500e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC-000"))

	msg = o.NewNewOrderMsg(add, genOrderID(add, 0, ctx, am), 1, "BTC-000_BNB", 355e8, 1e8)
	res, e = testClient.CheckTxSync(msg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(145e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(355e8), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC-000"))

	// using acc2

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 2, "BTC-000_BNB", 355e8, 250e8)
	res, e = testClient.CheckTxSync(msg, testApp.Codec)
	assert.NotEqual(uint32(0), res.Code)
	assert.Nil(e)
	assert.Regexp(".*do not have enough token to lock.*", res.GetLog())
	assert.Equal(int64(500e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 2, "BTC-000_BNB", 355e8, 200e8)
	res, e = testClient.CheckTxSync(msg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(500e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(200e8), GetLocked(ctx, add2, "BTC-000"))
}

func Test_handleNewOrder_DeliverTx(t *testing.T) {
	assert := assert.New(t)
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{})
	ctx := testApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	tradingPair := types.NewTradingPair("BTC-000", "BNB", 1e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, tradingPair)
	testApp.DexKeeper.AddEngine(tradingPair)

	add := Account(0).GetAddress()
	oid := fmt.Sprintf("%X-0", add)
	msg := o.NewNewOrderMsg(add, oid, 1, "BTC-000_BNB", 355e8, 1e8)

	res, e := testClient.DeliverTxSync(msg, testApp.Codec)
	t.Logf("res is %v and error is %v", res, e)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(utils.Fixed8(355e8), buys[0].price)
	assert.Equal(utils.Fixed8(1e8), buys[0].qty)
}

func Test_Match(t *testing.T) {
	assert := assert.New(t)
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{})
	ctx := testApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	ethPair := types.NewTradingPair("ETH-000", "BNB", 97e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, ethPair)
	testApp.DexKeeper.AddEngine(ethPair)
	btcPair := types.NewTradingPair("BTC-000", "BNB", 96e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	// setup accounts
	am := testApp.AccountKeeper
	acc := Account(0)
	acc2 := Account(1)
	acc3 := Account(2)
	add := acc.GetAddress()
	add2 := acc2.GetAddress()
	add3 := acc3.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	/*	--------------------------------------------------------------
		SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
		1500           102      300    300    300          -1200
		1500           101             300    300          -1200
		1500           100      100    400    400          -1100
		1500           99       200    600    600          -900
		1500   250     98       300    900    900          -600
		1250   250     97              900    900          -350
		1000   1000    96              900    900          -100*
	*/
	t.Log(GetAvail(ctx, add, "BTC-000"))
	t.Log(GetAvail(ctx, add, "BNB"))
	msg := o.NewNewOrderMsg(add, genOrderID(add, 0, ctx, am), 1, "BTC-000_BNB", 102e8, 300e8)
	res, e := testClient.DeliverTxSync(msg, testApp.Codec)
	t.Log(GetAvail(ctx, add, "BTC-000"))
	t.Log(GetAvail(ctx, add, "BNB"))
	msg = o.NewNewOrderMsg(add, genOrderID(add, 1, ctx, am), 1, "BTC-000_BNB", 100e8, 100e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	t.Log(GetAvail(ctx, add, "BTC-000"))
	t.Log(GetAvail(ctx, add, "BNB"))

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 2, "BTC-000_BNB", 96e8, 1000e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 1, ctx, am), 2, "BTC-000_BNB", 97e8, 250e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 2, ctx, am), 2, "BTC-000_BNB", 98e8, 250e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add, genOrderID(add, 2, ctx, am), 1, "BTC-000_BNB", 99e8, 200e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	t.Logf("res is %v and error is %v", res, e)
	msg = o.NewNewOrderMsg(add, genOrderID(add, 3, ctx, am), 1, "BTC-000_BNB", 98e8, 300e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))
	testApp.DexKeeper.MatchAndAllocateAll(ctx, nil)
	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(96e8), lastPx)
	assert.Equal(4, len(trades))
	// total execution is 900e8 BTC-000 @ price 96e8, notional is 86400e8, fee is 43.2e8 BNB
	assert.Equal(sdk.Coins{sdk.NewCoin("BNB", 86.4e8)}, fees.Pool.BlockFees().Tokens)
	assert.Equal(int64(100900e8), GetAvail(ctx, add, "BTC-000"))
	assert.Equal(int64(13556.8e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(98500e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(186356.8e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(600e8), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))

	// test ETH-000_BNB pair
	/*	--------------------------------------------------------------
		SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
		110            102      30     30     30           -80
		110            101      10     40     40           -70
		110            100             40     40           -70
		110            99       50     90     90           -20
		110    10      98              90     90           -20
		100    50      97              90     90           -10*
		50             96       15     105    50           55
		50     50      95              105    50           55
	*/

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 3, ctx, am), 1, "ETH-000_BNB", 102e8, 30e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 4, ctx, am), 1, "ETH-000_BNB", 101e8, 10e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 0, ctx, am), 2, "ETH-000_BNB", 95e8, 50e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 1, ctx, am), 2, "ETH-000_BNB", 98e8, 10e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 2, ctx, am), 2, "ETH-000_BNB", 97e8, 50e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 5, ctx, am), 1, "ETH-000_BNB", 96e8, 15e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 6, ctx, am), 1, "ETH-000_BNB", 99e8, 50e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	t.Logf("res is %v and error is %v", res, e)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))
	buys, sells = getOrderBook("ETH-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	testApp.DexKeeper.MatchAndAllocateAll(ctx, nil)
	buys, sells = getOrderBook("ETH-000_BNB")
	t.Logf("buys: %v", buys)
	t.Logf("sells: %v", sells)
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))
	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))
	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("ETH-000_BNB")
	assert.Equal(int64(97e8), lastPx)
	assert.Equal(4, len(trades))
	// total execution is 90e8 ETH @ price 97e8, notional is 8730e8
	// fee for this round is 8.73e8 BNB, totalFee is 95.13e8 BNB
	assert.Equal(sdk.Coins{sdk.NewCoin("BNB", 95.13e8)}, fees.Pool.BlockFees().Tokens)
	fees.Pool.Clear()
	assert.Equal(int64(100900e8), GetAvail(ctx, add, "BTC-000"))
	assert.Equal(int64(13556.8e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(98500e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(600e8), GetLocked(ctx, add2, "BTC-000"))
	// for buy, still locked = 15*96=1440, spent 8730
	// so reserve 1440+8730 = 10170
	// fee is 4.365e8 BNB
	assert.Equal(int64(176182.435e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(1440e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(100090e8), GetAvail(ctx, add2, "ETH-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "ETH-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(108725.635e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))
	assert.Equal(int64(99890e8), GetAvail(ctx, add3, "ETH-000"))
	assert.Equal(int64(20e8), GetLocked(ctx, add3, "ETH-000"))
	fees.Pool.Clear()
}

func Test_handleCancelOrder_CheckTx(t *testing.T) {
	assert := assert.New(t)
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{})
	ctx := testApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	tradingPair := types.NewTradingPair("BTC-000", "BNB", 1e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, tradingPair)
	testApp.DexKeeper.AddEngine(tradingPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	// setup accounts
	add := Account(0).GetAddress()
	oid := fmt.Sprintf("%X-0", add)
	add2 := Account(1).GetAddress()

	msg := o.NewCancelOrderMsg(add, "BTC-000_BNB", "DOESNOTEXIST-0")
	res, e := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.Regexp(".*Failed to find order \\[DOESNOTEXIST-0\\].*", res.GetLog())
	assert.Nil(e)
	newMsg := o.NewNewOrderMsg(add, oid, 1, "BTC-000_BNB", 355e8, 1e8)
	res, e = testClient.DeliverTxSync(newMsg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(145e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(355e8), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC-000"))
	msg = o.NewCancelOrderMsg(add2, "BTC-000_BNB", oid)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.Regexp(".*does not belong to transaction sender.*", res.GetLog())
	msg = o.NewCancelOrderMsg(add, "BTC-000_BNB", oid)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(500e8-2e4), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC-000"))
}

// #1: 20 orders, cancel twice in the middle, one in current block, one in next block
func Test_Cancel_1(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	orderMsgs := make([]o.NewOrderMsg, 20)
	for i := 0; i < len(orderMsgs); i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
		orderMsgs[i] = msg
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(20, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99790e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(210e8), GetLocked(ctx, add0, "BNB"))

	msgC := o.NewCancelOrderMsg(add0, "BTC-000_BNB", orderMsgs[10].Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(19, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99800.9998e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(199e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	msgC = o.NewCancelOrderMsg(add0, "BTC-000_BNB", orderMsgs[9].Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(18, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99810.9996e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(189e8), GetLocked(ctx, add0, "BNB"))
}

// #2: 10 orders, cancel the 1st inserted in next block
func Test_Cancel_2(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	orderMsgs := make([]o.NewOrderMsg, 10)
	for i := 0; i < len(orderMsgs); i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
		orderMsgs[i] = msg
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(10, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	msgC := o.NewCancelOrderMsg(add0, "BTC-000_BNB", orderMsgs[0].Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(9, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99945.9998e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(54e8), GetLocked(ctx, add0, "BNB"))
}

// #3: 16 orders, cancel the last inserted in next block
func Test_Cancel_3(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	orderMsgs := make([]o.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
		orderMsgs[i] = msg
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(16, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99864e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(136e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	msgC := o.NewCancelOrderMsg(add0, "BTC-000_BNB", orderMsgs[15].Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(15, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99879.9998e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(120e8), GetLocked(ctx, add0, "BNB"))
}

// #4: 16 orders, all inserted in one block, cancel all in next block
func Test_Cancel_4(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	orderMsgs := make([]o.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
		orderMsgs[i] = msg
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(16, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99864e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(136e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	for _, orderMsg := range orderMsgs {
		msgC := o.NewCancelOrderMsg(add0, "BTC-000_BNB", orderMsg.Id)
		_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9968e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
}

// #5: 16 orders, all inserted in different blocks, cancel all in next block
func Test_Cancel_5(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	orderMsgs := make([]o.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
		orderMsgs[i] = msg

		testClient.cl.EndBlockSync(ty.RequestEndBlock{})
		ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: int64(i+1)}).WithVoteInfos([]abci.VoteInfo{
			{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
		})
		testApp.DeliverState.Ctx = ctx
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(16, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99864e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(136e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 16}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	for _, orderMsg := range orderMsgs {
		msgC := o.NewCancelOrderMsg(add0, "BTC-000_BNB", orderMsg.Id)
		_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9968e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
}

// #6: 16 orders, all partially filled, all cancelled in next block
func Test_Cancel_6(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	acc1 := Account(1)
	add0 := acc0.GetAddress()
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	orderMsgs := make([]o.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", 1e8, 2e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
		orderMsgs[i] = msg
	}

	msgS := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 16e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99968e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(32e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99984e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(16e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1e8), lastPx)
	assert.Equal(16, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100016e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99967.992e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(16e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99984e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100015.992e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))

	for _, orderMsg := range orderMsgs {
		msgC := o.NewCancelOrderMsg(add0, "BTC-000_BNB", orderMsg.Id)
		_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100016e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99983.992e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99984e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100015.992e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #7: only one order exists on one side, cancel it in next block
func Test_Cancel_7(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	msgB := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	msgC := o.NewCancelOrderMsg(add0, "BTC-000_BNB", msgB.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))

	msgS := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	_, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))
	assert.Equal(int64(99999e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add0, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 2}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	msgC = o.NewCancelOrderMsg(add0, "BTC-000_BNB", msgS.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	if err != nil {
		panic(err)
	}

	_, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9996e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BTC-000"))
}

// #1: one IOC order, (either buy or sell), no fill, expire in next block
func Test_IOC_1(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	msgB := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))

	msgS := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 1e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	_, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))
	assert.Equal(int64(99999e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 2}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	_, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #2: numbers of IOC orders (either buy or sell), no fill, expire in next block
func Test_IOC_2(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	for i := 0; i < 5; i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		msg.TimeInForce = 3
		_, err = testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	buys, _ := getOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, add0, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, _ = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9995e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))

	for i := 0; i < 5; i++ {
		msg := o.NewNewOrderMsg(add1, genOrderID(add1, int64(i), ctx, am), 2, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		msg.TimeInForce = 3
		_, err = testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	_, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(5, len(sells))
	assert.Equal(int64(99985e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 2}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	_, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9995e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BTC-000"))
}

// #3: one IOC buy order, one IOC sell order, partial fill, expire in next block
func Test_IOC_3(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	acc1 := Account(1)
	add0 := acc0.GetAddress()
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	msgB := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 1e8, 2e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 1e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99998e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(2e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100001e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99998.9995e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000.9995e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #4: numbers of IOC orders: 1 full fill, 1 partial fill, 3 no fill and expire in next block
func Test_IOC_4(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	/*
	sum    sell    price    buy    sum    exec    imbal
	10             5        6      6      6	      -4
	10             4*       5      11	  10      1
	10             3        4      15     10      5
	10             2	    3	   18	  10	  8
	10     10      1        2      20     10      10
	*/

	for i := 0; i < 5; i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+2)*1e8)
		msg.TimeInForce = 3
		_, err = testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	msgB := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 10e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99930e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(70e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(4e8), lastPx)
	assert.Equal(2, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100010e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99959.9797e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100039.9800e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #5: numbers of GTE & IOC orders: GTE full fill, IOC partial fill and expire in next block
func Test_IOC_5(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	/*
	sum    sell    price    buy    sum    exec    imbal
	20             6        7      7      7       -14
	20             5        6      13     13	  -7
	20             4        5      18	  18      -2
	20             3*       20     38     20      18
	20     10      2               38     20      18
	10     10      1               38     20      28
	*/

	msgB1 := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 6e8, 7e8)
	_, err = testClient.DeliverTxSync(msgB1, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgB2 := o.NewNewOrderMsg(add0, genOrderID(add0, 1, ctx, am), 1, "BTC-000_BNB", 5e8, 6e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgB3 := o.NewNewOrderMsg(add0, genOrderID(add0, 2, ctx, am), 1, "BTC-000_BNB", 4e8, 5e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgB4 := o.NewNewOrderMsg(add0, genOrderID(add0, 3, ctx, am), 1, "BTC-000_BNB", 3e8, 10e8)
	msgB4.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB4, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgB5 := o.NewNewOrderMsg(add0, genOrderID(add0, 4, ctx, am), 1, "BTC-000_BNB", 3e8, 10e8)
	msgB5.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB5, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS1 := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS2 := o.NewNewOrderMsg(add1, genOrderID(add1, 1, ctx, am), 2, "BTC-000_BNB", 1e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(2, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99848e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(152e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(20e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(6, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100020e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99939.9700e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100059.9700e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #1: one buy order, one sell order in one block, full fill, (GTE-GTE, IOC-IOC, GTE-IOC)
func Test_Fill_1(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	msgB := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100001e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99998.9995e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000.9995e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))

	msgB = o.NewNewOrderMsg(add0, genOrderID(add0, 1, ctx, am), 1, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS = o.NewNewOrderMsg(add1, genOrderID(add1, 1, ctx, am), 2, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100001e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99997.9995e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99998e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000.9995e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 2}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100002e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99997.9990e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99998e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100001.9990e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))

	msgB = o.NewNewOrderMsg(add0, genOrderID(add0, 2, ctx, am), 1, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS = o.NewNewOrderMsg(add1, genOrderID(add1, 2, ctx, am), 2, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100002e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99996.9990e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100001.9990e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 3}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100003e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99996.9985e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100002.9985e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #2: one big order fills the other side
// it is covered in Test_Match_And_Allocation

// #3: one big IOC order fills the other side (GTE & IOC), and expire in next block
func Test_Fill_3(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	for i := 0; i < 5; i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		if i % 2 == 0 {
			msg.TimeInForce = 3
		}
		_, err = testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	msgS := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 100e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99900e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1e8), lastPx)
	assert.Equal(5, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100015e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99984.9925e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100014.9925e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #4: all orders (GTE & IOC) come in the same block and fully filled each other
func Test_Fill_4(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	/*
	sum    sell    price    buy    sum    exec    imbal
	6              5        3      3      3       -3
	6              4        2      5      5       -1
	6      3       3*       1      6	  6       0
	3      2       2               6      3       3
	1      1       1	           6      1       5
	*/

	for i := 0; i < 3; i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", int64(i+3)*1e8, int64(i+1)*1e8)
		if i % 1 == 0 {
			msg.TimeInForce = 3
		}
		_, err = testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	for i := 0; i < 3; i++ {
		msg := o.NewNewOrderMsg(add1, genOrderID(add1, int64(i), ctx, am), 2, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		if i % 2 == 0 {
			msg.TimeInForce = 3
		}
		_, err = testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(3, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99974e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(26e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99994e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(6e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(4, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100006e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99981.9910e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99994e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100017.9910e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #5: all orders (GTE & IOC) come in the same block and left 3 orders (from same users) partially filled in proportion
func Test_Fill_5(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	t.Log("lotSize:", btcPair.LotSize)
	t.Log("tickSize:", btcPair.TickSize)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	/*
	sum    sell    price    buy    sum    exec    imbal
	22             3*       30     30	  22      8
	22     7       2               30     22      8
	15     15      1	           30     15      15
	*/

	for i := 0; i < 3; i++ {
		msg := o.NewNewOrderMsg(add0, genOrderID(add0, int64(i), ctx, am), 1, "BTC-000_BNB", 3e8, 10e8)
		_, err = testClient.DeliverTxSync(msg, testApp.Codec)
		if err != nil {
			panic(err)
		}
	}

	msgS1 := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 15e8)
	msgS1.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS2 := o.NewNewOrderMsg(add1, genOrderID(add1, 1, ctx, am), 2, "BTC-000_BNB", 2e8, 7e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99910e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(22e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(4, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100022e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99909.967e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(24e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100065.9670e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

// #6: all orders (GTE & IOC) come in the same block and left 3 orders (from diff users) partially filled in proportion
func Test_Fill_6(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	t.Log("lotSize:", btcPair.LotSize)
	t.Log("tickSize:", btcPair.TickSize)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	acc2 := Account(2)
	add2 := acc2.GetAddress()
	acc3 := Account(3)
	add3 := acc3.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	/*
	sum    sell    price    buy    sum    exec    imbal
	22             3*       30     30	  22      8
	22     7       2               30     22      8
	15     15      1	           30     15      15
	*/

	msgB1 := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 3e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB1, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgB2 := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 1, "BTC-000_BNB", 3e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgB3 := o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 1, "BTC-000_BNB", 3e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS1 := o.NewNewOrderMsg(add3, genOrderID(add3, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 15e8)
	msgS1.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS2 := o.NewNewOrderMsg(add3, genOrderID(add3, 1, ctx, am), 2, "BTC-000_BNB", 2e8, 7e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, add1, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(22e8), GetLocked(ctx, add3, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(4, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	// TODO: it will fail in a random fashion, because of the sort of the addr
	assert.Equal(int64(100007.3334e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(9996998899990), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(7.9998e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(100007.3333e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(9996998900005), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(8.0001e8), GetLocked(ctx, add1, "BNB"))
	assert.Equal(int64(100007.3333e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(9996998900005), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(8.0001e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(100065.9670e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BTC-000"))
}

// #7: buy & sell orders get filled in zig-zag way
func Test_Fill_7(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 0}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	t.Log("lotSize:", btcPair.LotSize)
	t.Log("tickSize:", btcPair.TickSize)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	add0 := acc0.GetAddress()
	acc1 := Account(1)
	add1 := acc1.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	// trade @ 10

	msgB := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	msgS := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 10e8, 5e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99900e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99995e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100005e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99899.975e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99995e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100049.975e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))

	/* trade @ 9.5
	sum    sell    price    buy    sum    exec    imbal
	10             10       5      5      5	      -5
	10     10      8	           5      5       -5
	*/

	msgS = o.NewNewOrderMsg(add1, genOrderID(add1, 1, ctx, am), 2, "BTC-000_BNB", 8e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100005e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99899.975e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100049.975e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 2}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9.5e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100010e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99902.45125e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100097.45125e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, add1, "BTC-000"))

	/* trade @ 9
    sum    sell    price    buy    sum    exec    imbal
    5              9        10     10     5	      5
    5      5       8	           10     5       5
    */

	msgB = o.NewNewOrderMsg(add0, genOrderID(add0, 1, ctx, am), 1, "BTC-000_BNB", 9e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100010e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99812.45125e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100097.45125e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 3}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100015e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99812.42875e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(45e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100142.42875e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))

	/* trade @ 8.55
    sum    sell    price    buy    sum    exec    imbal
    10             9        5      5      5	      -5
    10     10      5	           5      5       -5
    */

	msgS = o.NewNewOrderMsg(add1, genOrderID(add1, 2, ctx, am), 2, "BTC-000_BNB", 5e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100015e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99812.42875e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(45e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100142.42875e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 4}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8.55e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100020e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99814.657375e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100185.157375e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, add1, "BTC-000"))

	/* trade @ 8.9775
    sum    sell    price    buy    sum    exec    imbal
    5              12       10     10     5	      5
    5      5       5	           10     5       5
    */

	msgB = o.NewNewOrderMsg(add0, genOrderID(add0, 2, ctx, am), 1, "BTC-000_BNB", 12e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	if err != nil {
		panic(err)
	}

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(100020e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99694.657375e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(120e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100185.157375e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, add1, "BTC-000"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 5}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8.9775e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(100025e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(9970974743125), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, add0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(10023002243125), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
}

//

func Test_Match_And_Allocation(t *testing.T) {
	assert := assert.New(t)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{Address: addr}
	tokens := []tokens.GenesisToken{{"BNB","BNB",100000000e8,addr,false}}
	appAcc := &common.AppAccount{baseAcc,"baseAcc",sdk.Coins(nil),sdk.Coins(nil)}

	valAddr := ed25519.GenPrivKey().PubKey().Address()
	genaccs := make([]GenesisAccount, 1)
	genaccs[0] = NewGenesisAccount(appAcc, valAddr)

	genesisState := GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(testApp.Codec, genesisState)
	if err != nil {
		panic(err)
	}

	testApp.InitChain(abci.RequestInitChain{Validators: []abci.ValidatorUpdate{}, AppStateBytes: stateBytes})

	ctx := testApp.DeliverState.Ctx
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 1}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC-000_BNB")
	btcPair := types.NewTradingPair("BTC-000", "BNB", 10e8)
	t.Log("lotSize:", btcPair.LotSize)
	t.Log("tickSize:", btcPair.TickSize)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeManager.UpdateConfig(newTestFeeConfig())

	am := testApp.AccountKeeper
	acc0 := Account(0)
	acc1 := Account(1)
	acc2 := Account(2)
	acc3 := Account(3)
	add0 := acc0.GetAddress()
	add1 := acc1.GetAddress()
	add2 := acc2.GetAddress()
	add3 := acc3.GetAddress()
	ResetAccounts(ctx, testApp, 100000e8, 100000e8, 100000e8)

	//#1 cannot buy with more than they have
	msg1_1 := o.NewNewOrderMsg(add0, genOrderID(add0, 0, ctx, am), 1, "BTC-000_BNB", 1e8, 75000e8)
	res, err := testClient.DeliverTxSync(msg1_1, testApp.Codec)

	msg1_2 := o.NewNewOrderMsg(add0, genOrderID(add0, 1, ctx, am), 1, "BTC-000_BNB", 1e8, 75000e8)
	res, err = testClient.DeliverTxSync(msg1_2, testApp.Codec)

	assert.Equal(true, strings.Contains(res.Log, "do not have enough token to lock"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(25000e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(75000e8), GetLocked(ctx, add0, "BNB"))

	//#2 cannot sell more than they have
	msg2_1 := o.NewNewOrderMsg(add1, genOrderID(add1, 0, ctx, am), 2, "BTC-000_BNB", 1e8, 60000e8)
	res, err = testClient.DeliverTxSync(msg2_1, testApp.Codec)

	msg2_2 := o.NewNewOrderMsg(add1, genOrderID(add1, 1, ctx, am), 2, "BTC-000_BNB", 1e8, 60000e8)
	res, err = testClient.DeliverTxSync(msg2_2, testApp.Codec)

	assert.Equal(true, strings.Contains(res.Log, "do not have enough token to lock"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(40000e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(60000e8), GetLocked(ctx, add1, "BTC-000"))

	//#3 cancel will return fund
	msg3_1 := o.NewCancelOrderMsg(add0, "BTC-000_BNB", msg1_1.Id)
	res, err = testClient.DeliverTxSync(msg3_1, testApp.Codec)
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, add0, "BNB"))

	msg3_2 := o.NewNewOrderMsg(add0, genOrderID(add0, 2, ctx, am), 1, "BTC-000_BNB", 1e8, 75000e8)
	res, err = testClient.DeliverTxSync(msg3_2, testApp.Codec)
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(24999.9998e8), GetAvail(ctx, add0, "BNB"))
	assert.Equal(int64(75000e8), GetLocked(ctx, add0, "BNB"))

	msg3_3 := o.NewCancelOrderMsg(add0, "BTC-000_BNB", msg3_2.Id)
	res, err = testClient.DeliverTxSync(msg3_3, testApp.Codec)
	assert.Equal(int64(100000e8), GetAvail(ctx, add0, "BTC-000"))
	assert.Equal(int64(99999.9996e8), GetAvail(ctx, add0, "BNB"))

	msg3_4 := o.NewCancelOrderMsg(add1, "BTC-000_BNB", msg2_1.Id)
	res, err = testClient.DeliverTxSync(msg3_4, testApp.Codec)
	assert.Equal(int64(100000e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, add1, "BNB"))

	//#4, test different price and allocation rules

	/*
	[- min surplus (absolute leftover volume)]
	sum    sell    price    buy    sum    exec    imbal
	150            12       30     30     30	  -120
	150		       11              30	  30      -120
	150		       10       10     40     40      -110
	150            9	    20	   60	  60	  -90
	150	   25	   8	    30	   90	  90	  -60
	125	   25	   7		       90     90	  -35
	100	   100	   6		       90	  90	  -10
	*/

	msg := o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 1, "BTC-000_BNB", 12e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 1, ctx, am), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 2, ctx, am), 1, "BTC-000_BNB", 9e8, 20e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 3, ctx, am), 1, "BTC-000_BNB", 8e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 0, ctx, am), 2, "BTC-000_BNB", 8e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 1, ctx, am), 2, "BTC-000_BNB", 7e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 2, ctx, am), 2, "BTC-000_BNB", 6e8, 100e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells := getOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 2}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(6e8), lastPx)
	assert.Equal(4, len(trades))

	assert.Equal(int64(100090e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(99459.73e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(100539.73e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

	/*
	incoming
    buy 15 @ 12
    sell 10 @ 6
	[- min surplus (absolute leftover volume)
	 - orders from earlier are filled first]
	sum    sell    price    buy    sum    exec    imbal
	70             12       15	   15	  15      -55
	70  		   11              15	  15	  -55
	70		       10		       15	  15	  -55
	70		       9		       15	  15      -55
	70	   25	   8		       15	  15      -55
	45	   25	   7		       15	  15	  -30
	20	   20	   6		       15	  15      -5
	*/
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 4, ctx, am), 1, "BTC-000_BNB", 12e8, 15e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add1, genOrderID(add1, 1, ctx, am), 2, "BTC-000_BNB", 6e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 3}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(6e8), lastPx)
	assert.Equal(2, len(trades))

	assert.Equal(int64(100105e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(99369.6850e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(100599.7e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100029.9848e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, add1, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BNB"))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 4}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	/*
	incoming
    buy 5 @ 12
    buy 5 @ 10
    buy 10 @ 8
    buy 5 @ 7
	[- max matched volume]
	sum    sell    price    buy    sum    exec    imbal
    55		       12	    5	   5	  5	      -50
    55		       11		       5	  5	      -50
    55		       10	    5	   10	  10	  -45
    55		       9		       10	  10	  -45
    55	   25	   8	    10	   20	  20	  -35
    30	   25	   7	    5	   25	  25	  -5
    5	   5	   6		       25	  5	      20
	*/

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 5, ctx, am), 1, "BTC-000_BNB", 12e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 6, ctx, am), 1, "BTC-000_BNB", 10e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 7, ctx, am), 1, "BTC-000_BNB", 8e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 8, ctx, am), 1, "BTC-000_BNB", 7e8, 5e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 5}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(4, len(trades))

	assert.Equal(int64(100130e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(99194.5975e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(100739.63e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, add1, "BTC-000"))
	assert.Equal(int64(100064.9673e8), GetAvail(ctx, add1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add1, "BNB"))

	/*
	incoming
	buy 10 @ 13
    buy 10 @ 10
    sell 25 @ 8
	[- adjust market pressure, sell side, ap > rp - 5% => lowest]
	sum    sell    price    buy    sum    exec    imbal
    30		       13	    10	   10	  10	  -20
    30		       12		       10	  10      -20
    30		       11		       10	  10	  -20
    30		       10	    10	   20	  20	  -10
    30		       9		       20	  20	  -10
    30	   25	   8		       20     20	  -10
    5	   5	   7		       20	  5	      15
	*/
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 9, ctx, am), 1, "BTC-000_BNB", 13e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 10, ctx, am), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 3, ctx, am), 2, "BTC-000_BNB", 8e8, 25e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 6}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100150e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(99034.5175e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99825e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(100899.55e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

	/*
	incoming
    buy 10 @ 13
    buy 10 @ 10
    sell 10 @ 11
	[- adjust market pressure, sell side, ap > rp - 5% => lowest]
	sum    sell    price    buy    sum    exec    imbal
    45		       13	    10	   10	  10	  -35
    45		       12		       10	  10	  -35
    45	   10	   11		       10	  10	  -35
    35	           10	    10	   20	  10	  -15
    35		       9		       20	  10	  -15
    35	   35	   8		       20	  10	  -15
	*/
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 11, ctx, am), 1, "BTC-000_BNB", 13e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 4, ctx, am), 2, "BTC-000_BNB", 11e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 12, ctx, am), 1, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 7}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(2, len(trades))

	assert.Equal(int64(100170e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(98874.4375e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99815e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(101059.47e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

	/*
	incoming
    buy 10 @ 12
    buy 10 @ 9
    buy 10 @ 9
	[- adjust market pressure, buy side, else: min(rp + 5%, highest)
	 - orders with same price and block height are split proportionally]
	sum    sell    price    buy    sum    exec    imbal
    25		       12	    10	   10	  10	  -15
    25	   10	   11		       10	  10	  -15
    15		       10		       10	  10	  -5
    15		       9	    20	   20	  15	  5
    15	   15	   8		       20	  15	  5
	*/

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 13, ctx, am), 1, "BTC-000_BNB", 12e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 14, ctx, am), 1, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 15, ctx, am), 1, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(2, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 8}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8.4e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100185e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(98613.3745e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99815e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(101185.4070e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

	/*
	incoming
    buy 50 @ 20
    sell 10 @ 10
    sell 10 @ 9
    sell 10 @ 7
	[- adjust market pressure, buy side, ap > rp + 5% => lowest]
	big order (up)
	sum    sell    price    buy    sum    exec    imbal
    40		       20	    50	   50	  40	  10
    40		       12		       50	  40	  10
    40	   10	   11		       50	  40	  10
    30	   10	   10		       50	  30	  20
    20	   10	   9	    15	   65	  20	  45
	10	           8	           65	  10	  55
    10	   10	   7		       65	  10	  55
	*/
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 16, ctx, am), 1, "BTC-000_BNB", 20e8, 50e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 5, ctx, am), 2, "BTC-000_BNB", 10e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 6, ctx, am), 2, "BTC-000_BNB", 9e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 7, ctx, am), 2, "BTC-000_BNB", 7e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(4, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 9}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(4, len(trades))

	assert.Equal(int64(100225e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(97973.1545e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(335e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99785e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(101625.1870e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

	/*
	incoming
    buy 30 @ 15
    sell 10 @ 17
    sell 10 @ 15
    sell 30 @ 9
	[- min surplus (absolute leftover volume)]
	big order (up)
	sum    sell    price    buy    sum    exec    imbal
    50		       20	    10	   10	  10	  -40
    50	   10	   17		       10	  10	  -40
    40	   10	   15	    30	   40	  30	  0
    30		       12		       40	  30	  10
    30		       11		       40	  30	  10
    30		       10		       40	  30	  10
    30	   30	   9	    15	   55	  15	  25
	*/

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 8, ctx, am), 2, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 17, ctx, am), 1, "BTC-000_BNB", 15e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 9, ctx, am), 2, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 10, ctx, am), 2, "BTC-000_BNB", 9e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 10}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(15e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100265e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(97572.8545e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99735e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(102224.8870e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

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
    60	   10	   17	    10	   10	  10	  -50
    50	   10	   16		       10	  10	  -40
    40	   10	   15	    10	   20	  20	  -20
    30	           12		       20	  20	  -10
    30		       11	    10	   30	  30	  0
    30		       10		       30	  30	  0
    30	   30	   9	    15	   45	  30	  15
	*/

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 18, ctx, am), 1, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 11, ctx, am), 2, "BTC-000_BNB", 16e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 19, ctx, am), 1, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 12, ctx, am), 2, "BTC-000_BNB", 15e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 20, ctx, am), 1, "BTC-000_BNB", 11e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 13, ctx, am), 2, "BTC-000_BNB", 9e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(4, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 11}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100295e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(97242.6895e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99685e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(102554.7220e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

	/*
	incoming
    buy 30 @ 17
	[- max matched volume]
	big order (up)
	sum    sell    price    buy    sum    exec    imbal
    30	   10	   17	    30	   30	  30	  0
    20	   10	   16		       30     20	  10
	10	   10	   15		       30	  10	  20
    0	           12		       30	  0	      30
    0		       11		       30	  0	      30
    0		       10		       30	  0	      30
    0		       9	    15	   45	  0	      45
	*/

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 21, ctx, am), 1, "BTC-000_BNB", 17e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(3, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 12}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(17e8), lastPx)
	assert.Equal(3, len(trades))

	assert.Equal(int64(100325e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(96732.4345e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99685e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(103064.4670e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))

	/*
	incoming
    buy 10 @ 17
    buy 10 @ 12
    sell 30 @ 10
	[ - adjust market pressure, sell side, ap < rp - 5% => highest]
	big order (down)
	sum    sell    price    buy    sum    exec    imbal
    30		       17	    10	   10	  10	  -20
    30		       16		       10	  10	  -20
    30		       15		       10	  10	  -20
    30		       12	    10	   20	  20	  -10
    30		       11		       20	  20	  -10
    30	   30	   10		       20	  20	  -10
    0		       9	    15	   35	  0	      35
	*/

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 22, ctx, am), 1, "BTC-000_BNB", 17e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 23, ctx, am), 1, "BTC-000_BNB", 12e8, 10e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 14, ctx, am), 2, "BTC-000_BNB", 10e8, 30e8)
	res, err = testClient.DeliverTxSync(msg, testApp.Codec)

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(ty.RequestEndBlock{})
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: valAddr, Height: 13}).WithVoteInfos([]abci.VoteInfo{
		{Validator: abci.Validator{Address: valAddr, Power: 10}, SignedLastBlock: true},
	})
	testApp.DeliverState.Ctx = ctx

	buys, sells = getOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(12e8), lastPx)
	assert.Equal(2, len(trades))

	assert.Equal(int64(100345e8), GetAvail(ctx, add2, "BTC-000"))
	assert.Equal(int64(96492.3145e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC-000"))
	assert.Equal(int64(135e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(99655e8), GetAvail(ctx, add3, "BTC-000"))
	assert.Equal(int64(103304.3470e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, add3, "BTC-000"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))
}