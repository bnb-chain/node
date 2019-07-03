package apptest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/order"
)

// note that maker orders are marked as x(m) in order book

// 1 maker order @ sell; many taker orders @ sell and buy;
// maker order partially filled;
// maker order limit price = concluded price;
// 1 taker order limit price < maker order limit price;
// maker order qty > taker order qty;
func Test_Maker_Sell_1a(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   150            12       30     30     30      -120
	   150            11              30     30      -120
	   150            10       10     40     40      -110
	   150            9        20     60     60      -90
	   150    25      8        30     90     90      -60
	   125    100(m)  7               90     90      -35
	   25     25      6               90     90      65
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 100e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 12e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99120e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(880e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(150e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	// TODO: confirm that maker order is no longer matched at first in this case
	// 1> maker order limit price = concluded price
	// 2> taker order at maker side has better price
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[0].TickType)
	assert.Equal(int64(0.0875e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0175e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0175e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0350e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0350e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(20e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0700e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0700e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(30e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.1050e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.1050e8), trades[4].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99369.6850e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99850e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100629.6850e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, addr1, "BTC-000"))
}

// 1 maker order @ sell; many taker orders @ sell and buy;
// maker order partially filled;
// maker order limit price = concluded price;
// 1 taker order limit price < maker order limit price;
// maker order qty < taker order qty;
func Test_Maker_Sell_1b(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   119            12       30     30     30      -89
	   119            11              30     30      -89
	   119            10       10     40     40      -89
	   119            9        20     60     60      -59
	   119    25      8        30     90     90      -29
	   94     35(m)   7               90     90      -4
	   59     59      6               90     90      31
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 35e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 12e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 59e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99120e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(880e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99881e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(119e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(30e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[0].TickType)
	assert.Equal(int64(0.1050e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.1050e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[1].TickType)
	assert.Equal(int64(0.0350e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0350e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(19e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[2].TickType)
	assert.Equal(int64(0.0665e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0665e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(1e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0035e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0035e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(30e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.1050e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.1050e8), trades[4].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99369.6850e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99881e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100629.6850e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(29e8), GetLocked(ctx, addr1, "BTC-000"))
}

// 1 maker order @ sell; many taker orders @ sell and buy;
// maker order fully filled;
// maker order limit price < concluded price;
// maker order is matched using its limited price;
// taker order is matched using the concluded price;
// change bnb fee config to produce fees with 8+ decimal points
func Test_Maker_Sell_1c(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   125            12       30     30     30      -95
	   125            11              30     30      -95
	   125            10       10     40     40      -85
	   125            9        20     60     60      -65
	   125    25      8        30     90     90      -35
	   100    25      7               90     90      -10
	   75     75(m)   6               90     90      15
	*/

	addr, ctx, accs := SetupTest_new()

	// change the default test fee config
	testFeeConfig.FeeRateNative = 475
	testApp.DexKeeper.FeeManager.UpdateConfig(testFeeConfig)

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 75e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 12e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99120e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(880e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99875e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(125e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(8, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	// 75*30/90 = 25; + 0.0001 = 25.0001
	// 75*10/90 = 8.3333
	// 75*20/90 = 16.6666
	// 75*30/90 = 25
	assert.Equal(int64(6e8), trades[0].LastPx)
	assert.Equal(int64(25.0001e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	// 6*25.0001 = 0.071250285 => 0.07125028
	assert.Equal(int64(0.07125028e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.07125028e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(6e8), trades[1].LastPx)
	assert.Equal(int64(8.3333e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	// 6*8.3333 = 0.023749905 => 0.02374990
	assert.Equal(int64(0.02374990e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.02374990e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(6e8), trades[2].LastPx)
	assert.Equal(int64(16.6666e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.04749981e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.04749981e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(6e8), trades[3].LastPx)
	assert.Equal(int64(25e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.07125e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.07125e8), trades[3].SellerFee.Tokens[0].Amount)
	// 15*30/90 = 5; + 0.0001 = 5.0001; 30 - 25.0001 = 4.9999
	// 15*10/90 = 1.6666; 10 - 8.3333 = 1.6667
	// 15*20/90 = 3.3333; 20 - 16.6666 = 3.3334
	// 15*30/90 = 5; 30 - 25 = 5
	assert.Equal(int64(7e8), trades[4].LastPx)
	assert.Equal(int64(4.9999e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[4].TickType)
	// 7*4.9999 = 0.0166246675 => 0.01662466
	assert.Equal(int64(0.01662466e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.01662466e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[5].LastPx)
	assert.Equal(int64(1.6667e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[5].TickType)
	// 7*1.6667 = 0.0055417775 => 0.00554177
	assert.Equal(int64(0.00554177e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.00554177e8), trades[5].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[6].LastPx)
	assert.Equal(int64(3.3334e8), trades[6].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[6].TickType)
	// 7*3.3334 = 0.011083555 => 0.01108355
	assert.Equal(int64(0.01108355e8), trades[6].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.01108355e8), trades[6].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[7].LastPx)
	assert.Equal(int64(5e8), trades[7].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[7].TickType)
	assert.Equal(int64(0.016625e8), trades[7].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.016625e8), trades[7].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99444.73637503e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99875e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100554.73637503e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, addr1, "BTC-000"))

	// restore the default test fee config
	testFeeConfig.FeeRateNative = 500
	testApp.DexKeeper.FeeManager.UpdateConfig(testFeeConfig)
}

// 3 maker orders @ sell (same qty, 1 price level); many taker orders @ sell and buy;
// maker orders fully filled with 1 taker order
// maker orders qty: 25, 25, 25 (all came in same block from same user)
func Test_Maker_Sell_2a(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   100            10       75     75     75      -25
	   100            9               75     75      -25
	   100    25      8               75     75      -25
	   75     75(m)   7               75     75      0
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 75e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99250e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(750e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99900e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0875e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(25e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0875e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(25e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0875e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[2].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100075e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99474.7375e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99900e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100524.7375e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
}

// 3 maker orders @ sell (same qty, 1 price level); many taker orders @ sell and buy;
// maker orders fully filled with 1 taker order
// maker orders qty: 25, 25, 25 (all came in different blocks from different users)
func Test_Maker_Sell_2b(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   100            10       75     75     75      -25
	   100            9               75     75      -25
	   100    25      8               75     75      -25
	   75     75(m)   7               75     75      0
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 75e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99250e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(750e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99950e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0875e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(25e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0875e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(25e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0875e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[2].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100075e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99474.7375e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99950e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

// 3 maker orders @ sell (same qty, 1 price level); many taker orders @ sell and buy;
// maker orders fully filled with 3 taker orders (same qty, 1 price level)
// maker orders qty: 25, 25, 25 (all came in different blocks from different users)
// taker orders qty: 25, 25, 25 (from same users)
func Test_Maker_Sell_2c(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   100            10       75     75     75      -25
	   100            9               75     75      -25
	   100    25      8               75     75      -25
	   75     75(m)   7               75     75      0
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99250e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(750e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99950e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0875e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(25e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0875e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(25e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0875e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[2].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100075e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99474.7375e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99950e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

// 3 maker orders @ sell (same qty, 1 price level); many taker orders @ sell and buy;
// maker orders fully filled with 3 taker orders (same qty, 1 price level)
// make orders qty: 25, 25, 25 (all came in different blocks from different users)
// taker orders qty: 31, 31, 31 (from different users)
func Test_Maker_Sell_2d(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   118            10       93     93     93      -25
	   118            9               93     93      -25
	   118    25      8               93     93      -25
	   93     75(m)   7               93     93      0
	   18     18      6               93     18
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 18e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(93e8), buys[0].qty)
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99690e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(310e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99932e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(68e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99690e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(310e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99690e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(310e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(18e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.Neutral), trades[0].TickType)
	assert.Equal(int64(0.0630e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0630e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(13e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.04550e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.04550e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(12e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0420e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0420e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(19e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0665e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0665e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(6e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.0210e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0210e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(25e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[5].TickType)
	assert.Equal(int64(0.0875e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[5].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100031e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99782.8915e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99932e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100300.8495e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100174.9125e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100031e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99782.8915e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100031e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99782.8915e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// 3 maker orders @ sell (same qty, 1 price level); many taker orders @ sell and buy;
// maker orders partially filled
// maker orders qty: 25, 25, 25 (all came in same block from different users)
// taker order qty: 20, 20, 20 (from different users)
func Test_Maker_Sell_2e(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   125            10       60     60     60      -65
	   125            9               60     60      -65
	   125    25      8               60     60      -65
	   100    75(m)   7               60     60      -40
	   25     25      6               60     25      35
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(60e8), buys[0].qty)
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99800e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(75e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99800e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99800e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(20e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[0].TickType)
	assert.Equal(int64(0.0700e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0700e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[1].TickType)
	assert.Equal(int64(0.0175e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0175e8), trades[1].SellerFee.Tokens[0].Amount)
	// 60-25 = 35e8, it will be split by 3 users
	// 35/3 = 11.6666e8; given the lot size is 1e4
	// 35-11.6666*3 = 0.0002e8; so it's 2 lots
	// after the split:
	// user a: 11.6666+0.0001 = 11.6667
	// user b: 11.6666+0.0001 = 11.6667
	// user c: 11.6666
	assert.Equal(int64(11.6667e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.04083345e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.04083345e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(3.3333e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.01166655e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.01166655e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8.3334e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.02916690e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.02916690e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(11.6666e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[5].TickType)
	assert.Equal(int64(0.04083310e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.04083310e8), trades[5].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100020e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99859.9300e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100256.53856655e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(38.3333e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100081.62606655e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(13.3333e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100081.62536690e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(13.3334e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100020e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99859.9300e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100020e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99859.9300e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// 3 maker orders @ sell (same qty, 1 price level); many taker orders @ sell and buy;
// maker orders partially filled
// maker orders qty: 25, 25, 25 (all came in different blocks from different users)
// taker order qty: 20, 20, 20 (from different users)
func Test_Maker_Sell_2f(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   125            10       60     60     60      -65
	   125            9               60     60      -65
	   125    25      8               60     60      -65
	   100    75(m)   7               60     60      -40
	   25     25      6               60     25      35
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(60e8), buys[0].qty)
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99800e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(75e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99800e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99800e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(20e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[0].TickType)
	assert.Equal(int64(0.0700e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0700e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[1].TickType)
	assert.Equal(int64(0.0175e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0175e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(15e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0525e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0525e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0350e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0350e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.0350e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0350e8), trades[4].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100020e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99859.9300e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100349.8250e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100069.9650e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100020e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99859.9300e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100020e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99859.9300e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// 3 maker orders @ sell (diff qty, 1 price level); many taker orders @ sell and buy;
// maker orders fully filled with 3 taker order
// maker orders qty: 30, 35, 10 (all came in same block from different users)
// taker order qty: 11, 13, 51 (from different users)
func Test_Maker_Sell_3a(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   100            10       75     75     75      -25
	   100            9               75     75      -25
	   100    25      8               75     75      -25
	   75     75(m)   7               75     75      0
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 30e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 35e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 11e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 13e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 51e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99890e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(110e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99965e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99870e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(130e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99490e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(510e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(30e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.1050e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.1050e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(21e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0735e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0735e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(13e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0455e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0455e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(1e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0035e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0035e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.0350e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0350e8), trades[4].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100011e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99922.9615e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100209.8950e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99965e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100244.8775e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100069.9650e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100013e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99908.9545e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100051e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99642.8215e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// 3 maker orders @ sell (diff qty, 1 price level); many taker orders @ sell and buy;
// maker orders fully filled with 3 taker orders (same qty, 1 price level)
// maker orders qty: 30, 35, 10 (all came in different blocks from different users)
// taker orders qty: 11, 13, 51 (from different users)
func Test_Maker_Sell_3b(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   100            10       75     75     75      -25
	   100            9               75     75      -25
	   100    25      8               75     75      -25
	   75     75(m)   7               75     75      0
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 30e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 35e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 11e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 13e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 51e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99890e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(110e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99965e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99870e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(130e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99490e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(510e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(30e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.1050e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.1050e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(21e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0735e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0735e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(13e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0455e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0455e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(1e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0035e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0035e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.0350e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0350e8), trades[4].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100011e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99922.9615e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100209.8950e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99965e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100244.8775e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100069.9650e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100013e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99908.9545e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100051e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99642.8215e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// 3 maker orders @ sell (diff qty, 1 price level); many taker orders @ sell and buy;
// maker orders fully filled with 3 taker orders (same qty, 1 price level)
// maker order qty: 21, 24, 30 (all came in different blocks from different users)
// taker orders qty: 31, 31, 31 (from different users)
// maker order limit price < concluded price;
// maker order is matched using its limited price;
// taker order is matched using the concluded price;
func Test_Maker_Sell_3c(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 9.5
	   sum    sell    price    buy    sum    exec    imbal
	   100            10       93     93     93      -7
	   100            9               93     93      -7
	   100    25      8               93     93      -7
	   75     75(m)   7               93     93      18
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 21e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 24e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(93e8), buys[0].qty)
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99690e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(310e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99954e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(46e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99976e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(24e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99690e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(310e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99690e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(310e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9.5e8), lastPx)
	assert.Equal(8, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(7e8), trades[0].LastPx)
	assert.Equal(int64(21e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0735e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0735e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[1].LastPx)
	assert.Equal(int64(4e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0140e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0140e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[2].LastPx)
	assert.Equal(int64(20e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0700e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0700e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[3].LastPx)
	assert.Equal(int64(5e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0175e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0175e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[4].LastPx)
	assert.Equal(int64(25e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.0875e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(9.5e8), trades[5].LastPx)
	assert.Equal(int64(6e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[5].TickType)
	assert.Equal(int64(0.0285e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0285e8), trades[5].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(9.5e8), trades[6].LastPx)
	assert.Equal(int64(6e8), trades[6].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[6].TickType)
	assert.Equal(int64(0.0285e8), trades[6].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0285e8), trades[6].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(9.5e8), trades[7].LastPx)
	assert.Equal(int64(6e8), trades[7].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[7].TickType)
	assert.Equal(int64(0.0285e8), trades[7].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0285e8), trades[7].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100031e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99767.8840e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99954e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100317.8410e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(7e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99976e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100167.9160e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100209.8950e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100031e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99767.8840e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100031e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99767.8840e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// 3 maker orders @ sell (diff qty, 1 price level); many taker orders @ sell and buy;
// maker orders partially filled
// maker orders qty: 21, 24, 30 (all came in different blocks from different users)
// taker orders qty: 11, 27, 22 (from different users)
func Test_Maker_Sell_3d(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   125            10       60     60     60      -65
	   125            9               60     60      -65
	   125    25      8               60     60      -65
	   100    75(m)   7               60     60      -40
	   25     25      6               60     25      35
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 21e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 24e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 11e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 27e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 22e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(60e8), buys[0].qty)
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99890e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(110e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99929e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(71e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99976e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(24e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99730e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(270e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99780e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(220e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[0].TickType)
	assert.Equal(int64(0.0875e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(2e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0070e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0070e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(19e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0665e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0665e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(3e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0105e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0105e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(11e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.0385e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0385e8), trades[4].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100011e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99922.9615e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99929e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100321.8390e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99976e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100097.9510e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100027e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99810.9055e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100022e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99845.9230e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// multiple maker orders (all came in different blocks)
func Test_Maker_Sell_4(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   40             10       25     25     25      -15
	   40     5(m)    9               25     25      -15
	   35     20(m)   8        7      32     32      -3
	   15     5       7        9      41     15      26
	   10     5       6        10     51     10      41
	   5      5(m)    5        10     61     5       56
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 5e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 5e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 7e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 9e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 6e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 4, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 5e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 4, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(5, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99521e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(479e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99960e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(40e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	// 5*25/32 = 3.9062; + 0.0001 = 3.9063
	// 5*7/32 = 1.0937;
	assert.Equal(int64(5e8), trades[0].LastPx)
	assert.Equal(int64(3.9063e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.00976575e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.00976575e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(5e8), trades[1].LastPx)
	assert.Equal(int64(1.0937e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.00273425e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.00273425e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[2].LastPx)
	assert.Equal(int64(5e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[2].TickType)
	assert.Equal(int64(0.0200e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0200e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[3].LastPx)
	assert.Equal(int64(5e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[3].TickType)
	assert.Equal(int64(0.0200e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0200e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[4].LastPx)
	assert.Equal(int64(11.0937e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.04437480e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.04437480e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[5].LastPx)
	assert.Equal(int64(5.9063e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[5].TickType)
	assert.Equal(int64(0.02362520e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.02362520e8), trades[5].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100032e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99585.8795e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(173e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99960e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100240.8795e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(8e8), GetLocked(ctx, addr1, "BTC-000"))
}

// multiple maker orders (all came in same block)
// both fill and cancel
func Test_Maker_Sell_5(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   40             10       25     25     25      -15
	   40     5(m)    9               25     25      -15
	   35     20(m)   8        7      32     32      -3
	   15     5       7        9      41     15      26
	   10     5       6        10     51     10      41
	   5      5(m)    5        10     61     5       56
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 5e8)
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 5e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 7e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 9e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 6e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 4, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 5e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 4, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	msgC := order.NewCancelOrderMsg(addr1, "BTC-000_BNB", msgS.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(4, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99521e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(479e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99965e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	// 5*25/32 = 3.9062; + 0.0001 = 3.9063
	// 5*7/32 = 1.0937;
	assert.Equal(int64(5e8), trades[0].LastPx)
	assert.Equal(int64(3.9063e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.00976575e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.00976575e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(5e8), trades[1].LastPx)
	assert.Equal(int64(1.0937e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.00273425e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.00273425e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[2].LastPx)
	assert.Equal(int64(5e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[2].TickType)
	assert.Equal(int64(0.0200e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0200e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[3].LastPx)
	assert.Equal(int64(5e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[3].TickType)
	assert.Equal(int64(0.0200e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0200e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[4].LastPx)
	assert.Equal(int64(11.0937e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.04437480e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.04437480e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[5].LastPx)
	assert.Equal(int64(5.9063e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[5].TickType)
	assert.Equal(int64(0.02362520e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.02362520e8), trades[5].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100032e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99585.8795e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(173e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99965e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100240.8793e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(3e8), GetLocked(ctx, addr1, "BTC-000"))
}

// IOC taker order: buy 25 @ 10, fully filled
// IOC taker order: buy 7 @ 8, partially filled
func Test_Maker_Sell_6(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   36             10       25     25     25      -11
	   36     5(m)    9               25     25      -11
	   31     20(m)   8        7      32     31      1
	   11     6       7               32     0       21
	   5      5       6               32     0       27
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 5e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	msg.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 7e8)
	msg.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 6e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(4, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99694e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(306e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99964e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(36e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(5e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[0].TickType)
	assert.Equal(int64(0.0200e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0200e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(6e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[1].TickType)
	assert.Equal(int64(0.024e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.024e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(14e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.0560e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0560e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(6e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.0240e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0240e8), trades[3].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100031e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99751.8760e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99964e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100247.8760e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))
}

// maker orders match priority at different price levels
func Test_Maker_Sell_7(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   36             10       15     15     15      -21
	   36     5       9               15     15      -21
	   31     20(m)   8        7      22     22      -9
	   11     6(m)    7               22     22      10
	   5      5(m)    6               22     22      17
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 20e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 6e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(3, len(sells))

	ctx = UpdateContextC(addr, ctx, 4)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 15e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 7e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(4, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99794e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(206e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99964e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(36e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	// 5*15/22 = 3.4090; + 0.0001 = 3.4091
	// 5*7/22 = 1.5909
	assert.Equal(int64(6e8), trades[0].LastPx)
	assert.Equal(int64(3.4091e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.01022730e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.01022730e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(6e8), trades[1].LastPx)
	assert.Equal(int64(1.5909e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.00477270e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.00477270e8), trades[1].SellerFee.Tokens[0].Amount)
	// 6*15/22 = 4.0909; + 0.0001 = 4.0910
	// 6*7/22 = 1.9090;
	assert.Equal(int64(7e8), trades[2].LastPx)
	assert.Equal(int64(4.0910e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[2].TickType)
	assert.Equal(int64(0.01431850e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.01431850e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(7e8), trades[3].LastPx)
	assert.Equal(int64(1.9090e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[3].TickType)
	assert.Equal(int64(0.00668150e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.00668150e8), trades[3].SellerFee.Tokens[0].Amount)
	// 11*15/22 = 7.5; 15-3.4091-4.0910=7.4999
	// 11*7/22 = 3.5; 7-1.5909-1.9090=3.5001
	assert.Equal(int64(8e8), trades[4].LastPx)
	assert.Equal(int64(7.4999e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[4].TickType)
	assert.Equal(int64(0.02999960e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.02999960e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[5].LastPx)
	assert.Equal(int64(3.5001e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[5].TickType)
	assert.Equal(int64(0.01400040e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.01400040e8), trades[5].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100022e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99839.9200e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99964e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100159.9200e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(14e8), GetLocked(ctx, addr1, "BTC-000"))
}

// maker side taker orders match priority at different price level
// sell 20 @ 8 -> 2 orders: 15, 5
func Test_Maker_Sell_8(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   35             10       29     29     29      -6
	   35     5       9               29     29      -6
	   30     20      8               29     29      -1
	   10     5(m)    7               29     10      19
	   5      5       6               29     5       24
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 5e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 29e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 8e8, 15e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(4, len(sells))
	assert.Equal(utils.Fixed8(20e8), sells[2].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99710e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(290e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(20e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, addr2, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(7e8), trades[0].LastPx)
	assert.Equal(int64(5e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0175e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0175e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[1].LastPx)
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[1].TickType)
	assert.Equal(int64(0.0200e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0200e8), trades[1].SellerFee.Tokens[0].Amount)
	// 19/20 = 0.95
	// 15*0.95 = 14.25
	// 5*0.95 = 4.75
	assert.Equal(int64(8e8), trades[2].LastPx)
	assert.Equal(int64(4.75e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[2].TickType)
	assert.Equal(int64(0.0190e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0190e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(8e8), trades[3].LastPx)
	assert.Equal(int64(14.25e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[3].TickType)
	assert.Equal(int64(0.0570e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0570e8), trades[3].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100029e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99772.8865e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100112.9435e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5.25e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100113.943e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0.75e8), GetLocked(ctx, addr2, "BTC-000"))
}

// one taker order @ maker side has higher price than the maker order
// one taker order @ maker side has lower price than the maker order
// one taker order @ maker side has same price as the maker order
// maker order limited price < concluded price
func Test_Maker_Sell_9a(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 10
	   sum    sell      price    buy    sum    exec    imbal
	   26               10       29     29     26      3
	   26     1         9               29     26      3
	   25     20(3m,17) 8               29     25      4
	   5                7               29     5       24
	   5      5         6               29     5       24
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 3e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 29e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 9e8, 1e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 1, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 8e8, 17e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 2, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(3, len(sells))
	assert.Equal(utils.Fixed8(20e8), sells[1].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99710e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(290e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(3e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99977e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(23e8), GetLocked(ctx, addr2, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(8e8), trades[0].LastPx)
	assert.Equal(int64(3e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0120e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0120e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[1].LastPx)
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[1].TickType)
	assert.Equal(int64(0.0250e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0250e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[2].LastPx)
	assert.Equal(int64(17e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[2].TickType)
	assert.Equal(int64(0.0850e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0850e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[3].LastPx)
	assert.Equal(int64(1e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[3].TickType)
	assert.Equal(int64(0.0050e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0050e8), trades[3].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100026e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99715.873e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100023.988e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99977e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100229.8850e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
}

// one taker order @ maker side has lower price than the maker order
// one taker order @ maker side has same price as the maker order
// maker order limited price = concluded price
func Test_Maker_Sell_9b(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell       price    buy    sum    exec    imbal
	   90     65(10m,55) 7        90     90     90      0
	   25     25         6               90     90      65
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr1, 0, ctx)
	msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 10e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	ctx = UpdateContextC(addr, ctx, 2)

	oid = GetOrderId(addr0, 0, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 90e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 55e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 1, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 6e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))
	assert.Equal(utils.Fixed8(65e8), sells[1].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99370e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(630e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99920e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(80e8), GetLocked(ctx, addr2, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.Neutral), trades[0].TickType)
	assert.Equal(int64(0.0875e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0875e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(10e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[1].TickType)
	assert.Equal(int64(0.0350e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0350e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(55e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.Neutral), trades[2].TickType)
	assert.Equal(int64(0.1925e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.1925e8), trades[2].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99369.6850e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100069.9650e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99920e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100559.7200e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
}

// no maker order, multiple taker orders
func Test_Maker_Sell_9c(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 9
	   sum    sell      price    buy         sum    exec    imbal
	   26               10       24 (2,19,3) 24     24
	   26     1         9        5           29     26      3
	   25     20(3,17)  8                    29     25
	   5                7                    29     5
	   5      5         6                    29     5
	*/

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 5e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 2e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 19e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 3e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 3e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 9e8, 1e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 1, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 8e8, 17e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 2, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(utils.Fixed8(24e8), buys[0].qty)
	assert.Equal(3, len(sells))
	assert.Equal(utils.Fixed8(20e8), sells[1].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99715e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(285e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(3e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99977e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(23e8), GetLocked(ctx, addr2, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(7, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	// trades order: no sort @ sell; sort @ buy
	assert.Equal(int64(5e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[0].TickType)
	assert.Equal(int64(0.0225e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0225e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(3e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[1].TickType)
	assert.Equal(int64(0.0135e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0135e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(11e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[2].TickType)
	assert.Equal(int64(0.0495e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0495e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(3e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[3].TickType)
	assert.Equal(int64(0.0135e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0135e8), trades[3].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(2e8), trades[4].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[4].TickType)
	assert.Equal(int64(0.0090e8), trades[4].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0090e8), trades[4].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(1e8), trades[5].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[5].TickType)
	assert.Equal(int64(0.0045e8), trades[5].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0045e8), trades[5].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(1e8), trades[6].LastQty)
	assert.Equal(int8(matcheng.BuySurplus), trades[6].TickType)
	assert.Equal(int64(0.0045e8), trades[6].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0045e8), trades[6].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100026e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99738.883e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(27e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100026.9865e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99977e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100206.8965e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
}

/*
test #10a: cancel no filled maker order @ sell side
*/
func Test_Maker_Sell_10a_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 2, "BTC-000_BNB", 7e8, 100e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))

	assert.Equal(int64(99900e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr0, "BTC-000"))

	ctx = UpdateContextC(addr, ctx, 2)

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msg.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BTC-000"))
}

/*
test #10b: cancel partial filled maker order @ sell side
*/
func Test_Maker_Sell_10b_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidS := GetOrderId(addr0, 0, ctx)
	msgS := order.NewNewOrderMsg(addr0, oidS, 2, "BTC-000_BNB", 7e8, 100e8)
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))

	assert.Equal(int64(99900e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr0, "BTC-000"))

	ctx = UpdateContextC(addr, ctx, 2)

	oidB := GetOrderId(addr1, 0, ctx)
	msgB := order.NewNewOrderMsg(addr1, oidB, 1, "BTC-000_BNB", 7e8, 50e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(1, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))

	assert.Equal(int64(99900e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100349.8250e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100050e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99649.8250e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))

	ctx = UpdateContextC(addr, ctx, 3)

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msgS.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))

	assert.Equal(int64(99950e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100349.8250e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100050e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99649.8250e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))
}
