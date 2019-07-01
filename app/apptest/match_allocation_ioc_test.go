package apptest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/order"
)

/*
test #1: one IOC order, either buy or sell, no fill, expire in next block
*/
func Test_IOC_1(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 1e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))

	assert.Equal(int64(99999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2: numbers of IOC orders, either buy or sell, no fill, expire in next block
*/
func Test_IOC_2(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		msg.TimeInForce = 3
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9995e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr1, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		msg.TimeInForce = 3
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(sells))

	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9995e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BTC-000"))
}

/*
test #3: one IOC buy order, one IOC sell order, partial fill, expire in next block
*/
func Test_IOC_3(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 2e8)
	msgB.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 1e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(2e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100001e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99998.9995e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000.9995e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #4: numbers of IOC orders: 1 full fill, 1 partial fill, 3 no fill and expire in next block
*/
func Test_IOC_4(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	/*
		sum    sell    price    buy    sum    exec    imbal
		10             5        6      6      6	      -4
		10             4*       5      11	  10      1
		10             3        4      15     10      5
		10             2	    3	   18	  10	  8
		10     10      1        2      20     10      10
	*/

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+2)*1e8)
		msg.TimeInForce = 3
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 10e8)
	msgS.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99930e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(70e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(4e8), lastPx)
	assert.Equal(2, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100010e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99959.9797e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100039.9800e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #5: numbers of GTE & IOC orders: GTE full fill, IOC partial fill and expire in next block
*/
func Test_IOC_5(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	/*
		sum    sell    price    buy    sum    exec    imbal
		20             6        7      7      7       -14
		20             5        6      13     13	  -7
		20             4        5      18	  18      -2
		20             3*       20     38     20      18
		20     10      2               38     20      18
		10     10      1               38     20      28
	*/

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 6e8, 7e8)
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidB2 := GetOrderId(addr0, 1, ctx)
	msgB2 := order.NewNewOrderMsg(addr0, oidB2, 1, "BTC-000_BNB", 5e8, 6e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidB3 := GetOrderId(addr0, 2, ctx)
	msgB3 := order.NewNewOrderMsg(addr0, oidB3, 1, "BTC-000_BNB", 4e8, 5e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	assert.NoError(err)

	oidB4 := GetOrderId(addr0, 3, ctx)
	msgB4 := order.NewNewOrderMsg(addr0, oidB4, 1, "BTC-000_BNB", 3e8, 10e8)
	msgB4.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB4, testApp.Codec)
	assert.NoError(err)

	oidB5 := GetOrderId(addr0, 4, ctx)
	msgB5 := order.NewNewOrderMsg(addr0, oidB5, 1, "BTC-000_BNB", 3e8, 10e8)
	msgB5.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB5, testApp.Codec)
	assert.NoError(err)

	oidS1 := GetOrderId(addr1, 0, ctx)
	msgS1 := order.NewNewOrderMsg(addr1, oidS1, 2, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	assert.NoError(err)

	oidS2 := GetOrderId(addr1, 1, ctx)
	msgS2 := order.NewNewOrderMsg(addr1, oidS2, 2, "BTC-000_BNB", 1e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(utils.Fixed8(20e8), buys[3].qty)
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99848e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(152e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(20e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(6, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100020e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99939.9700e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100059.9700e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #6: expire fee is larger than the bnb balance, the bnb balance becomes 0
*/
func Test_IOC_6(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest(1e12)
	addr0 := accs[0].GetAddress()
	ResetAccount(ctx, addr0, 1e2, 100000e8, 100000e8)

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e7, 10)
	msgB.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msgB.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
}

/*
test #7: no bnb balance, expire fee is charged in the balance of the opposite token
*/