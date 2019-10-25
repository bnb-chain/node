package apptest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/order"
)

/*
test #1a: 3 consecutive matches, split 1, 1, 10 (3 orders with same price) from same block
*/
func Test_Split_1a(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidB2 := GetOrderId(addr1, 0, ctx)
	msgB2 := order.NewNewOrderMsg(addr1, oidB2, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidB3 := GetOrderId(addr2, 0, ctx)
	msgB3 := order.NewNewOrderMsg(addr2, oidB3, 1, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr3, 0, ctx)
	msgS := order.NewNewOrderMsg(addr3, oidS, 2, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(12e8), buys[0].qty)
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(2e8), lastPx)
	assert.Equal(3, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(2e8), buys[0].qty)
	assert.Equal(0, len(sells))

	assert.Equal(int64(10000083340000), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(9999799916660), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0.3332e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(10000083330000), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(9999799916670), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0.3334e8), GetLocked(ctx, addr1, "BNB"))
	assert.Equal(int64(10000833330000), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(9997999166670), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(3.3334e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100019.99e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

/*
test #1b: 3 consecutive matches, split 1, 1, 10 (3 orders with same price) from same block, lot size test case 1
*/
func Test_Split_1b(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest(1e5)
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidB2 := GetOrderId(addr1, 0, ctx)
	msgB2 := order.NewNewOrderMsg(addr1, oidB2, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidB3 := GetOrderId(addr2, 0, ctx)
	msgB3 := order.NewNewOrderMsg(addr2, oidB3, 1, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr3, 0, ctx)
	msgS := order.NewNewOrderMsg(addr3, oidS, 2, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(12e8), buys[0].qty)
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(2e8), lastPx)
	assert.Equal(3, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(2e8), buys[0].qty)
	assert.Equal(0, len(sells))

	assert.Equal(int64(100001e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99997.9990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(100001e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99997.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))
	assert.Equal(int64(100008e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99979.9920e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(4e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100019.99e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

/*
test #1c: 3 consecutive matches, split 1, 1, 10 (3 orders with same price) from same block, lot size test case 2
*/
func Test_Split_1c(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest(1e7)
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidS := GetOrderId(addr3, 0, ctx)
	msgS := order.NewNewOrderMsg(addr3, oidS, 2, "BTC-000_BNB", 2e8, 1e7)
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	_, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	testApp.DexKeeper.UpdateLotSize("BTC-000_BNB", 1e8)

	ctx = UpdateContextC(addr, ctx, 2)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidB2 := GetOrderId(addr1, 0, ctx)
	msgB2 := order.NewNewOrderMsg(addr1, oidB2, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidB3 := GetOrderId(addr2, 0, ctx)
	msgB3 := order.NewNewOrderMsg(addr2, oidB3, 1, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(12e8), buys[0].qty)
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(2e8), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(119000e4), buys[0].qty)
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000.1000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99997.9999e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1.8000e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99998e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(2e8), GetLocked(ctx, addr1, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(20e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99999.9000e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000.1999e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

/*
test #2: 3 consecutive matches, split 1, 1, 10 (3 orders with same price) from different blocks
*/
func Test_Split_2(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 2)

	oidB2 := GetOrderId(addr1, 0, ctx)
	msgB2 := order.NewNewOrderMsg(addr1, oidB2, 1, "BTC-000_BNB", 2e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 3)

	oidB3 := GetOrderId(addr2, 0, ctx)
	msgB3 := order.NewNewOrderMsg(addr2, oidB3, 1, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgB3, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextC(addr, ctx, 4)

	oidS := GetOrderId(addr3, 0, ctx)
	msgS := order.NewNewOrderMsg(addr3, oidS, 2, "BTC-000_BNB", 2e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(12e8), buys[0].qty)
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(2e8), lastPx)
	assert.Equal(3, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(2e8), buys[0].qty)
	assert.Equal(0, len(sells))

	assert.Equal(int64(100001e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99997.9990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(100001e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99997.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BNB"))
	assert.Equal(int64(100008e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99979.9920e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(4e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100019.99e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

// lot size changed (e.g. due to significant price changes) lead to not enough balance, chain panic
func Test_Split_3(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest(1e8)
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	// the original lotsize is 1e5

	ctx = UpdateContextC(addr, ctx, 1)

	oidS1 := GetOrderId(addr0, 0, ctx)
	msgS1 := order.NewNewOrderMsg(addr0, oidS1, 2, "BTC-000_BNB", 1e4, 2e8)
	res, err := testClient.DeliverTxSync(msgS1, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	oidS2 := GetOrderId(addr1, 0, ctx)
	msgS2 := order.NewNewOrderMsg(addr1, oidS2, 2, "BTC-000_BNB", 1e4, 20e8)
	res, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	oidS3 := GetOrderId(addr2, 0, ctx)
	msgS3 := order.NewNewOrderMsg(addr2, oidS3, 2, "BTC-000_BNB", 1e4, 12e8)
	res, err = testClient.DeliverTxSync(msgS3, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	oidB1 := GetOrderId(addr3, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr3, oidB1, 1, "BTC-000_BNB", 1e4, 10e8)
	res, err = testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(utils.Fixed8(10e8), buys[0].qty)
	assert.Equal(utils.Fixed8(34e8), sells[0].qty)

	testApp.DexKeeper.UpdateLotSize("BTC-000_BNB", 1e9)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	// panic: no enough locked tokens to unlock,
	// oid: 7E4FF4423F1CDF142E915A7A08247381280E78DB-0,
	// newLocked: -800000000BTC-000, unlock: 1000000000

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1e4), lastPx)
	assert.Equal(2, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(utils.Fixed8(24e8), sells[0].qty)

	assert.Equal(int64(99998e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100000.00019990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000.00079960e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(12e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99988e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(12e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100010e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99999.99899950e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}

func Test_Split_4(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest(10)
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContextC(addr, ctx, 1)

	oidB := GetOrderId(addr0, 0, ctx)
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1, 1e12)
	res, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	oidS1 := GetOrderId(addr1, 0, ctx)
	msgS1 := order.NewNewOrderMsg(addr1, oidS1, 2, "BTC-000_BNB", 1, 1e12)
	res, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	oidS3 := GetOrderId(addr2, 0, ctx)
	msgS3 := order.NewNewOrderMsg(addr2, oidS3, 2, "BTC-000_BNB", 1, 1e12)
	res, err = testClient.DeliverTxSync(msgS3, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	oidS4 := GetOrderId(addr3, 0, ctx)
	msgS4 := order.NewNewOrderMsg(addr3, oidS4, 2, "BTC-000_BNB", 1, 1e12)
	res, err = testClient.DeliverTxSync(msgS4, testApp.Codec)
	assert.NoError(err)
	fmt.Println(res)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(3e12), sells[0].qty)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(1), lastPx)
	assert.Equal(1, len(trades))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(utils.Fixed8(2e12), sells[0].qty)

	assert.Equal(int64(11000000000000), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(9999999989995), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(9000000000000), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(10000000009995), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(9000000000000), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(10000000000000), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(1000000000000), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(9000000000000), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(10000000000000), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(1000000000000), GetLocked(ctx, addr3, "BTC-000"))
}
