package apptest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/order"
)

/*
test #1: 20 orders, cancel twice in the middle, one in current block, one in next block
*/
func Test_Cancel_1_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	orderMsgs := make([]order.NewOrderMsg, 20)
	for i := 0; i < len(orderMsgs); i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		orderMsgs[i] = msg
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(20, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99790e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(210e8), GetLocked(ctx, addr0, "BNB"))

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", orderMsgs[10].Id)
	_, err := testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(19, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99800.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(199e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	msgC = order.NewCancelOrderMsg(addr0, "BTC-000_BNB", orderMsgs[9].Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(18, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99810.9996e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(189e8), GetLocked(ctx, addr0, "BNB"))
}

/*
test #2: 10 orders, cancel the 1st one
*/
func Test_Cancel_2_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	orderMsgs := make([]order.NewOrderMsg, 10)
	for i := 0; i < len(orderMsgs); i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		orderMsgs[i] = msg
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(10, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", orderMsgs[0].Id)
	_, err := testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(9, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99945.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(54e8), GetLocked(ctx, addr0, "BNB"))
}

/*
test #3: 16 orders, cancel the last one
*/
func Test_Cancel_3_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	orderMsgs := make([]order.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		orderMsgs[i] = msg
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(16, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99864e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(136e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", orderMsgs[15].Id)
	_, err := testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(15, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99879.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(120e8), GetLocked(ctx, addr0, "BNB"))
}

/*
test #4: 16 orders, all inserted in current block, all cancelled in next block
*/
func Test_Cancel_4_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	orderMsgs := make([]order.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		orderMsgs[i] = msg
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(16, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99864e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(136e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	for _, orderMsg := range orderMsgs {
		msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", orderMsg.Id)
		_, err := testClient.DeliverTxSync(msgC, testApp.Codec)
		assert.NoError(err)
	}

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9968e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
}

/*
test #5: 16 orders, all inserted in different blocks, all cancelled in next block
*/
func Test_Cancel_5_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()

	orderMsgs := make([]order.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		ctx = UpdateContextC(addr, ctx, int64(i+1))

		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, 1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		orderMsgs[i] = msg

		testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(16, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99864e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(136e8), GetLocked(ctx, addr0, "BNB"))

	ctx = UpdateContextC(addr, ctx, 17)

	for _, orderMsg := range orderMsgs {
		msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", orderMsg.Id)
		_, err := testClient.DeliverTxSync(msgC, testApp.Codec)
		assert.NoError(err)
	}

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9968e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
}

/*
test #6: 16 orders, all partially filled, and all cancelled in next block
*/
func Test_Cancel_6_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	orderMsgs := make([]order.NewOrderMsg, 16)
	for i := 0; i < len(orderMsgs); i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 1e8, 2e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		orderMsgs[i] = msg
	}

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 16e8)
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(32e8), buys[0].qty)
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99968e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(32e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99984e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(16e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1e8), lastPx)
	assert.Equal(16, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(16e8), buys[0].qty)
	assert.Equal(0, len(sells))

	assert.Equal(int64(100016e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99967.992e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(16e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99984e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100015.992e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))

	for _, orderMsg := range orderMsgs {
		msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", orderMsg.Id)
		_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
		assert.NoError(err)
	}

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100016e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99983.992e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99984e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100015.992e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #7: only one order exists on one side (either buy or sell), cancel it in next block
*/
func Test_Cancel_7_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 1e8)
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msgB.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))

	oidS := GetOrderId(addr0, 1, ctx)
	msgS := order.NewNewOrderMsg(addr0, oidS, 2, "BTC-000_BNB", 1e8, 1e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))

	assert.Equal(int64(99999e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9998e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr0, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	msgC = order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msgS.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9996e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BTC-000"))
}

/*
test #8: cancel fee is larger than the bnb balance, the bnb balance becomes 0
*/
func Test_Cancel_8_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new(1e12)
	addr0 := accs[0].GetAddress()
	ResetAccount(ctx, addr0, 100, 100000e8, 100000e8)

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e7, 10)
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
test #9: no bnb balance, cancel fee is charged in the balance of the opposite token
*/
func Test_Cancel_9_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new(1)
	addr0 := accs[0].GetAddress()
	ResetAccount(ctx, addr0, 0, 200000e8, 100000e8)

	ctx = UpdateContextC(addr, ctx, 1)

	oidS := GetOrderId(addr0, 0, ctx)
	msgS := order.NewNewOrderMsg(addr0, oidS, 2, "BTC-000_BNB", 1, 100000e8)
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetLocked(ctx, addr0, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	msgC := order.NewCancelOrderMsg(addr0, "BTC-000_BNB", msgS.Id)
	_, err = testClient.DeliverTxSync(msgC, testApp.Codec)
	assert.NoError(err)

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BTC-000"))
}
