package apptest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/order"
)

/*
test #1: one buy order, one sell order in one block, full fill, (GTE-GTE, IOC-IOC, GTE-IOC)
*/
func Test_Fill_1(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 1e8)
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100001e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99998.9995e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000.9995e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))

	oidB = GetOrderId(addr0, 1, ctx)
	msgB = order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS = GetOrderId(addr1, 1, ctx)
	msgS = order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100001e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99997.9995e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99998e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000.9995e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100002e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99997.9990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99998e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100001.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))

	oidB = GetOrderId(addr0, 2, ctx)
	msgB = order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS = GetOrderId(addr1, 2, ctx)
	msgS = order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100002e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99996.9990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100001.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 4)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100003e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99996.9985e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99997e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100002.9985e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2: one big IOC order fills the other side (GTE & IOC), and expire in next block
*/
func Test_Fill_2(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		if i % 2 == 0 {
			msg.TimeInForce = 3
		}
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 100e8)
	msgS.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99900e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1e8), lastPx)
	assert.Equal(5, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100015e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99984.9925e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100014.9925e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #3: all orders (GTE & IOC) come in the same block and fully filled each other
*/
func Test_Fill_3(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	/*
		sum    sell    price    buy    sum    exec    imbal
		6              5        3      3      3       -3
		6              4        2      5      5       -1
		6      3       3*       1      6	  6       0
		3      2       2               6      3       3
		1      1       1	           6      1       5
	*/

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 3; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+3)*1e8, int64(i+1)*1e8)
		if i % 1 == 0 {
			msg.TimeInForce = 3
		}
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	for i := 0; i < 3; i++ {
		oid := GetOrderId(addr1, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		if i % 2 == 0 {
			msg.TimeInForce = 3
		}
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99974e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(26e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99994e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(6e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(4, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100006e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99981.9910e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99994e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100017.9910e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #4: all orders (GTE & IOC) come in the same block and left 3 orders (from same users) partially filled in proportion
*/
func Test_Fill_4(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	/*
		sum    sell    price    buy    sum    exec    imbal
		22             3*       30     30	  22      8
		22     7       2               30     22      8
		15     15      1	           30     15      15
	*/

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 3; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 3e8, 10e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	oidS1 := GetOrderId(addr1, 0, ctx)
	msgS1 := order.NewNewOrderMsg(addr1, oidS1, 2, "BTC-000_BNB", 1e8, 15e8)
	msgS1.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgS1, testApp.Codec)
	assert.NoError(err)

	oidS2 := GetOrderId(addr1, 1, ctx)
	msgS2 := order.NewNewOrderMsg(addr1, oidS2, 2, "BTC-000_BNB", 2e8, 7e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(30e8), buys[0].qty)
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99910e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(22e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(4, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(8e8), buys[0].qty)
	assert.Equal(0, len(sells))

	assert.Equal(int64(100022e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99909.967e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(24e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100065.9670e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #5: all orders (GTE & IOC) come in the same block and left 3 orders (from diff users) partially filled in proportion
*/
func Test_Fill_5(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	/*
		sum    sell    price    buy    sum    exec    imbal
		22             3*       30     30	  22      8
		22     7       2               30     22      8
		15     15      1	           30     15      15
	*/

	ctx = UpdateContextC(addr, ctx, 1)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 3e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidB2 := GetOrderId(addr1, 0, ctx)
	msgB2 := order.NewNewOrderMsg(addr1, oidB2, 1, "BTC-000_BNB", 3e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidB3 := GetOrderId(addr2, 0, ctx)
	msgB3 := order.NewNewOrderMsg(addr2, oidB3, 1, "BTC-000_BNB", 3e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	assert.NoError(err)

	oidS1 := GetOrderId(addr3, 0, ctx)
	msgS1 := order.NewNewOrderMsg(addr3, oidS1, 2, "BTC-000_BNB", 1e8, 15e8)
	msgS1.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	assert.NoError(err)

	oidS2 := GetOrderId(addr3, 1, ctx)
	msgS2 := order.NewNewOrderMsg(addr3, oidS2, 2, "BTC-000_BNB", 2e8, 7e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(30e8), buys[0].qty)
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr1, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(22e8), GetLocked(ctx, addr3, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(4, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(8e8), buys[0].qty)
	assert.Equal(0, len(sells))

	assert.Equal(int64(100007.3334e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(9996998899990), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(7.9998e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(100007.3333e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(9996998900005), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(8.0001e8), GetLocked(ctx, addr1, "BNB"))
	assert.Equal(int64(100007.3333e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(9996998900005), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(8.0001e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100065.9670e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

/*
test #6: buy & sell orders get filled in the zig-zag way
*/
func Test_Fill_6(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	// trade @ 10

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 10e8, 5e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99900e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99995e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100005e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99899.975e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99995e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100049.975e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))

	/* trade @ 9.5
	sum    sell    price    buy    sum    exec    imbal
	10             10       5      5      5	      -5
	10     10      8	           5      5       -5
	*/

	oidS = GetOrderId(addr1, 1, ctx)
	msgS = order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 8e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100005e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99899.975e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(50e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100049.975e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9.5e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100010e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99902.45125e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100097.45125e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))

	/* trade @ 9
	   sum    sell    price    buy    sum    exec    imbal
	   5              9        10     10     5	      5
	   5      5       8	           10     5       5
	*/

	oidB = GetOrderId(addr0, 1, ctx)
	msgB = order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 9e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100010e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99812.45125e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100097.45125e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 4)

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100015e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99812.42875e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(45e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100142.42875e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))

	/* trade @ 8.55
	   sum    sell    price    buy    sum    exec    imbal
	   10             9        5      5      5	      -5
	   10     10      5	           5      5       -5
	*/

	oidS = GetOrderId(addr1, 2, ctx)
	msgS = order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 5e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100015e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99812.42875e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(45e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100142.42875e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 5)

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8.55e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100020e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99814.657375e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100185.157375e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))

	/* trade @ 8.9775
	   sum    sell    price    buy    sum    exec    imbal
	   5              12       10     10     5	      5
	   5      5       5	           10     5       5
	*/

	oidB = GetOrderId(addr0, 2, ctx)
	msgB = order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 12e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100020e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99694.657375e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(120e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100185.157375e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(5e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 6)

	trades, lastPx = testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8.9775e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100025e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(9970974743125), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99975e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(10023002243125), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}