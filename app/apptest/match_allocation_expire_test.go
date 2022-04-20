package apptest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/bnb-chain/node/common/utils"
	"github.com/bnb-chain/node/plugins/dex/order"
)

/*
test #1: all orders on one side expire, either buy or sell
*/
func Test_Expire_1(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))

	ctx = UpdateContextC(addr, ctx, 4)

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr1, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(sells))

	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	tNow = time.Now()

	ctx = UpdateContextB(addr, ctx, 5, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 6, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2a: the first 5 levels expire, for buy
*/
func Test_Expire_2a(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	buys, _ := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99945e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(55e8), GetLocked(ctx, addr0, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	oidB := GetOrderId(addr0, 5, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 11e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(6, len(buys))
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99899.9990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2b: the first 5 levels expire, for sell
*/
func Test_Expire_2b(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 5; i++ {
		oid := GetOrderId(addr1, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", int64(i+1)*1e8, int64(i+1)*1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(sells))

	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 5, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 11e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(6, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99900e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #3: orders in the middle levels expire
*/
func Test_Expire_3(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 3; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+2)*1e8, int64(i+1)*1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		if i == 2 {
			for j := 0; j < 2; j++ {
				oid := GetOrderId(addr0, int64(i+1+j), ctx)
				msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", int64(i+2)*1e8, int64(i+1)*1e8)
				_, err = testClient.DeliverTxSync(msg, testApp.Codec)
				assert.NoError(err)
			}
		}
	}

	for i := 0; i < 3; i++ {
		oid := GetOrderId(addr1, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", int64(i+5)*1e8, int64(i+1)*1e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
		if i == 2 {
			for j := 0; j < 2; j++ {
				oid := GetOrderId(addr1, int64(i+1+j), ctx)
				msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", int64(i+5)*1e8, int64(i+1)*1e8)
				_, err = testClient.DeliverTxSync(msg, testApp.Codec)
				assert.NoError(err)
			}
		}
	}

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(9e8), buys[0].qty)
	assert.Equal(utils.Fixed8(2e8), buys[1].qty)
	assert.Equal(utils.Fixed8(1e8), buys[2].qty)
	assert.Equal(utils.Fixed8(1e8), sells[0].qty)
	assert.Equal(utils.Fixed8(2e8), sells[1].qty)
	assert.Equal(utils.Fixed8(9e8), sells[2].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99956e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(44e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99988e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(12e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	oidB5 := GetOrderId(addr0, 5, ctx)
	msgB5 := order.NewNewOrderMsg(addr0, oidB5, 1, "BTC-000_BNB", 1e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB5, testApp.Codec)
	assert.NoError(err)

	oidB6 := GetOrderId(addr0, 6, ctx)
	msgB6 := order.NewNewOrderMsg(addr0, oidB6, 1, "BTC-000_BNB", 4.25e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB6, testApp.Codec)
	assert.NoError(err)

	oidS5 := GetOrderId(addr1, 5, ctx)
	msgS5 := order.NewNewOrderMsg(addr1, oidS5, 2, "BTC-000_BNB", 4.75e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS5, testApp.Codec)
	assert.NoError(err)

	oidS6 := GetOrderId(addr1, 6, ctx)
	msgS6 := order.NewNewOrderMsg(addr1, oidS6, 2, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS6, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	// for buy: price level is ordered from high to low
	assert.Equal(utils.Fixed8(10e8), buys[0].qty)
	assert.Equal(utils.Fixed8(9e8), buys[1].qty)
	assert.Equal(utils.Fixed8(2e8), buys[2].qty)
	assert.Equal(utils.Fixed8(1e8), buys[3].qty)
	assert.Equal(utils.Fixed8(10e8), buys[4].qty)
	// for sell: price level is ordered from low to high
	assert.Equal(utils.Fixed8(10e8), sells[0].qty)
	assert.Equal(utils.Fixed8(1e8), sells[1].qty)
	assert.Equal(utils.Fixed8(2e8), sells[2].qty)
	assert.Equal(utils.Fixed8(9e8), sells[3].qty)
	assert.Equal(utils.Fixed8(10e8), sells[4].qty)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(10e8), buys[0].qty)
	assert.Equal(utils.Fixed8(10e8), buys[1].qty)
	assert.Equal(utils.Fixed8(10e8), sells[0].qty)
	assert.Equal(utils.Fixed8(10e8), sells[1].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99947.4990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(52.5e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(20e8), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #4a: expire partial filled orders, for buy
*/
func Test_Expire_4a(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	for i := 0; i < 3; i++ {
		oid := GetOrderId(addr0, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 1e8, 10e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 15e8)
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(30e8), buys[0].qty)
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(15e8), buys[0].qty)
	assert.Equal(0, len(sells))

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

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
test #4b: expire partial filled orders, for sell
*/
func Test_Expire_4b(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 15e8)
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	for i := 0; i < 3; i++ {
		oid := GetOrderId(addr1, int64(i), ctx)
		msg := order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 1e8, 10e8)
		_, err := testClient.DeliverTxSync(msg, testApp.Codec)
		assert.NoError(err)
	}

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(30e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(15e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(30e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(utils.Fixed8(15e8), sells[0].qty)

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

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
test #5: IOC orders, either buy or sell sent in breath block should not expire
*/
func Test_Expire_5(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextB(addr, ctx, 1, tNow)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 1e8)
	msgB.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 2e8, 1e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #6: expire fee is larger than the bnb balance, the bnb balance becomes 0
*/
func Test_Expire_6(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest(1e12)
	addr0 := accs[0].GetAddress()
	ResetAccount(ctx, addr0, 1e2, 100000e8, 100000e8)

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

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

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
}

/*
test #7: no bnb balance, expire fee is charged in the balance of the opposite token
*/
