package app

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/utils"
	o "github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
)

type level struct {
	price utils.Fixed8
	qty   utils.Fixed8
}

func getOrderBook(pair string) ([]level, []level) {
	buys := make([]level, 0)
	sells := make([]level, 0)
	orderbooks := testApp.DexKeeper.GetOrderBook(pair, 5)
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

func genOrderID(add sdk.AccAddress, seq int64, ctx sdk.Context, am auth.AccountMapper) string {
	acc := am.GetAccount(ctx, add)
	if acc.GetSequence() != seq {
		err := acc.SetSequence(seq)
		if err != nil {
			panic(err)
		}
		am.SetAccount(ctx, acc)
	}
	oid := fmt.Sprintf("%s-%d", add.String(), seq)
	return oid
}

func Test_handleNewOrder_CheckTx(t *testing.T) {
	assert := assert.New(t)
	ctx := testApp.NewContext(true, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, types.NewTradingPair("BTC", "BNB", 1e8))

	am := testApp.AccountMapper
	acc := Account(0)
	acc2 := Account(1)
	add := acc.GetAddress()
	add2 := acc2.GetAddress()
	msg := o.NewNewOrderMsg(add, genOrderID(add, 0, ctx, am), 1, "BTC_BNB", 355e8, 100e8)
	res, e := testClient.CheckTxSync(msg, testApp.Codec)
	assert.NotEqual(uint32(0), res.Code)
	assert.Nil(e)
	assert.Regexp(".*do not have enough token to lock.*", res.GetLog())
	assert.Equal(int64(500e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC"))

	msg = o.NewNewOrderMsg(add, genOrderID(add, 0, ctx, am), 1, "BTC_BNB", 355e8, 1e8)
	res, e = testClient.CheckTxSync(msg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(145e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(355e8), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC"))

	// using acc2

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 2, "BTC_BNB", 355e8, 250e8)
	res, e = testClient.CheckTxSync(msg, testApp.Codec)
	assert.NotEqual(uint32(0), res.Code)
	assert.Nil(e)
	assert.Regexp(".*do not have enough token to lock.*", res.GetLog())
	assert.Equal(int64(500e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add2, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BTC"))

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 2, "BTC_BNB", 355e8, 200e8)
	res, e = testClient.CheckTxSync(msg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(500e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(0), GetAvail(ctx, add2, "BTC"))
	assert.Equal(int64(200e8), GetLocked(ctx, add2, "BTC"))
}

func Test_handleNewOrder_DeliverTx(t *testing.T) {
	assert := assert.New(t)
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{})
	ctx := testApp.NewContext(false, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC_BNB")
	tradingPair := types.NewTradingPair("BTC", "BNB", 1e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, tradingPair)
	testApp.DexKeeper.AddEngine(tradingPair)

	add := Account(0).GetAddress()
	oid := fmt.Sprintf("%s-0", add.String())
	msg := o.NewNewOrderMsg(add, oid, 1, "BTC_BNB", 355e8, 1e8)

	res, e := testClient.DeliverTxSync(msg, testApp.Codec)
	t.Logf("res is %v and error is %v", res, e)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	buys, sells := getOrderBook("BTC_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(utils.Fixed8(355e8), buys[0].price)
	assert.Equal(utils.Fixed8(1e8), buys[0].qty)
}

func Test_Match(t *testing.T) {
	assert := assert.New(t)
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{})
	ctx := testApp.NewContext(false, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC_BNB")
	ethPair := types.NewTradingPair("ETH", "BNB", 1e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, ethPair)
	testApp.DexKeeper.AddEngine(ethPair)
	btcPair := types.NewTradingPair("BTC", "BNB", 1e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, btcPair)
	testApp.DexKeeper.AddEngine(btcPair)
	testApp.DexKeeper.FeeConfig.SetFeeRateNative(ctx, 500)
	testApp.DexKeeper.FeeConfig.SetFeeRate(ctx, 1000)

	// setup accounts
	am := testApp.AccountMapper
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
	msg := o.NewNewOrderMsg(add, genOrderID(add, 0, ctx, am), 1, "BTC_BNB", 102e8, 300e8)
	res, e := testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add, genOrderID(add, 1, ctx, am), 1, "BTC_BNB", 100e8, 100e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 0, ctx, am), 2, "BTC_BNB", 96e8, 1000e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 1, ctx, am), 2, "BTC_BNB", 97e8, 250e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 2, ctx, am), 2, "BTC_BNB", 98e8, 250e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add, genOrderID(add, 2, ctx, am), 1, "BTC_BNB", 99e8, 200e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	t.Logf("res is %v and error is %v", res, e)
	msg = o.NewNewOrderMsg(add, genOrderID(add, 3, ctx, am), 1, "BTC_BNB", 98e8, 300e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	buys, sells := getOrderBook("BTC_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))
	ctx, code, e := testApp.DexKeeper.MatchAndAllocateAll(ctx, testApp.AccountMapper, nil)
	t.Logf("res is %v and error is %v", code, e)
	buys, sells = getOrderBook("BTC_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	trades, lastPx := testApp.DexKeeper.GetLastTrades("BTC_BNB")
	assert.Equal(int64(96e8), lastPx)
	assert.Equal(4, len(trades))
	// total execution is 900e8 BTC @ price 96e8, notional is 86400e8, fee is 43.2e8 BNB
	assert.Equal(sdk.Coins{sdk.NewCoin("BNB", 86.4e8)}, tx.Fee(ctx).Tokens)
	assert.Equal(int64(100900e8), GetAvail(ctx, add, "BTC"))
	assert.Equal(int64(13556.8e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(98500e8), GetAvail(ctx, add2, "BTC"))
	assert.Equal(int64(186356.8e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(600e8), GetLocked(ctx, add2, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "BNB"))

	// test ETH_BNB pair
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

	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 3, ctx, am), 1, "ETH_BNB", 102e8, 30e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 4, ctx, am), 1, "ETH_BNB", 101e8, 10e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 0, ctx, am), 2, "ETH_BNB", 95e8, 50e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 1, ctx, am), 2, "ETH_BNB", 98e8, 10e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add3, genOrderID(add3, 2, ctx, am), 2, "ETH_BNB", 97e8, 50e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 5, ctx, am), 1, "ETH_BNB", 96e8, 15e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	msg = o.NewNewOrderMsg(add2, genOrderID(add2, 6, ctx, am), 1, "ETH_BNB", 99e8, 50e8)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	t.Logf("res is %v and error is %v", res, e)
	buys, sells = getOrderBook("BTC_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))
	buys, sells = getOrderBook("ETH_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))
	ctx, code, e = testApp.DexKeeper.MatchAndAllocateAll(ctx, testApp.AccountMapper, nil)
	t.Logf("res is %v and error is %v", code, e)
	buys, sells = getOrderBook("ETH_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))
	buys, sells = getOrderBook("BTC_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))
	trades, lastPx = testApp.DexKeeper.GetLastTrades("ETH_BNB")
	assert.Equal(int64(97e8), lastPx)
	assert.Equal(4, len(trades))
	// total execution is 90e8 ETH @ price 97e8, notional is 8730e8
	// fee for this round is 8.73e8 BNB, totalFee is 95.13e8 BNB
	assert.Equal(sdk.Coins{sdk.NewCoin("BNB", 95.13e8)}, tx.Fee(ctx).Tokens)
	assert.Equal(int64(100900e8), GetAvail(ctx, add, "BTC"))
	assert.Equal(int64(13556.8e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(98500e8), GetAvail(ctx, add2, "BTC"))
	assert.Equal(int64(600e8), GetLocked(ctx, add2, "BTC"))
	// for buy, still locked = 15*96=1440, spent 8730
	// so reserve 1440+8730 = 10170
	// fee is 4.365e8 BNB
	assert.Equal(int64(176182.435e8), GetAvail(ctx, add2, "BNB"))
	assert.Equal(int64(1440e8), GetLocked(ctx, add2, "BNB"))
	assert.Equal(int64(100090e8), GetAvail(ctx, add2, "ETH"))
	assert.Equal(int64(0), GetLocked(ctx, add2, "ETH"))
	assert.Equal(int64(100000e8), GetAvail(ctx, add3, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BTC"))
	assert.Equal(int64(108725.635e8), GetAvail(ctx, add3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add3, "BNB"))
	assert.Equal(int64(99890e8), GetAvail(ctx, add3, "ETH"))
	assert.Equal(int64(20e8), GetLocked(ctx, add3, "ETH"))
}

func Test_handleCancelOrder_CheckTx(t *testing.T) {
	assert := assert.New(t)
	testClient.cl.BeginBlockSync(abci.RequestBeginBlock{})
	ctx := testApp.NewContext(false, abci.Header{})
	InitAccounts(ctx, testApp)
	testApp.DexKeeper.ClearOrderBook("BTC_BNB")
	tradingPair := types.NewTradingPair("BTC", "BNB", 1e8)
	testApp.DexKeeper.PairMapper.AddTradingPair(ctx, tradingPair)
	testApp.DexKeeper.AddEngine(tradingPair)

	// setup accounts
	add := Account(0).GetAddress()
	oid := fmt.Sprintf("%s-0", add.String())
	add2 := Account(1).GetAddress()

	msg := o.NewCancelOrderMsg(add, oid, "doesnotexist")
	res, e := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.Regexp(".*Failed to find order \\[doesnotexist\\].*", res.GetLog())
	assert.Nil(e)
	newMsg := o.NewNewOrderMsg(add, oid, 1, "BTC_BNB", 355e8, 1e8)
	res, e = testClient.DeliverTxSync(newMsg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(145e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(355e8), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC"))
	msg = o.NewCancelOrderMsg(add2, oid, oid)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.Regexp(".*does not belong to transaction sender.*", res.GetLog())
	msg = o.NewCancelOrderMsg(add, oid, oid)
	res, e = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.Equal(uint32(0), res.Code)
	assert.Nil(e)
	assert.Equal(int64(500e8), GetAvail(ctx, add, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BNB"))
	assert.Equal(int64(200e8), GetAvail(ctx, add, "BTC"))
	assert.Equal(int64(0), GetLocked(ctx, add, "BTC"))
}
