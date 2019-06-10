package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/order"
)

// note that maker orders are marked as x(m) in order book
// the scenario design is the opposite of sell side
// for detailed descrption, please refer to *maker_sell_test

func Test_Maker_Buy_1a(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 11
	   sum    sell    price    buy    sum    exec    imbal
	   90             12       25     25     25
	   90		      11       100(m) 125	 90      35
	   90	  30	  10       25     150    90      60
	   60     20      9	    	      150	 60
	   40	  10	  8	    	      150	 40
	   30	          7		          150    30
	   30	  30	  6		          150	 30
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 11e8, 100e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 12e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 10e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(4, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(98350e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1650e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99910e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(30e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastQty)
	assert.Equal(int64(20e8), trades[2].LastQty)
	assert.Equal(int64(5e8), trades[3].LastQty)
	assert.Equal(int64(25e8), trades[4].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(98374.5050e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(635e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99910e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100989.5050e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

func Test_Maker_Buy_1b(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 11
	   sum    sell    price    buy    sum    exec    imbal
	   90             12       59     59     59
	   90		      11       35(m)  94	 90      4
	   90	  30	  10       25     119    90      29
	   60     20      9	    	      119	 60
	   40	  10	  8	    	      119	 40
	   30	          7		          119    30
	   30	  30	  6		          119	 30
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 11e8, 35e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 12e8, 59e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 10e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(4, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(98657e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1343e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99910e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(30e8), trades[0].LastQty)
	assert.Equal(int64(1e8), trades[1].LastQty)
	assert.Equal(int64(9e8), trades[2].LastQty)
	assert.Equal(int64(20e8), trades[3].LastQty)
	assert.Equal(int64(30e8), trades[4].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(98715.5050e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(294e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99910e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100989.5050e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

// maker order limit price > concluded price;
func Test_Maker_Buy_1c(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 11
	   sum    sell    price    buy    sum    exec    imbal
	   90             12       75(m)  75     75
	   90		      11       25     100	 90      10
	   90	  30	  10       25     125    90      35
	   60     20      9	    	      125	 60
	   40	  10	  8	    	      125	 40
	   30	          7		          125    30
	   30	  30	  6		          125	 30
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 12e8, 75e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 11e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 10e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(4, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(98575e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(1425e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99910e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(11e8), lastPx)
	assert.Equal(8, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	// 75*30/90 = 25; + 0.0001 = 25.0001
	// 75*10/90 = 8.3333
	// 75*20/90 = 16.6666
	// 75*30/90 = 25
	assert.Equal(int64(12e8), trades[0].LastPx)
	assert.Equal(int64(25.0001e8), trades[0].LastQty)
	assert.Equal(int64(12e8), trades[1].LastPx)
	assert.Equal(int64(8.3333e8), trades[1].LastQty)
	assert.Equal(int64(12e8), trades[2].LastPx)
	assert.Equal(int64(16.6666e8), trades[2].LastQty)
	assert.Equal(int64(12e8), trades[3].LastPx)
	assert.Equal(int64(25e8), trades[3].LastQty)
	// 15*30/90 = 5; + 0.0001 = 5.0001; 30 - 25.0001 = 4.9999
	// 15*10/90 = 1.6666; 10 - 8.3333 = 1.6667
	// 15*20/90 = 3.3333; 20 - 16.6666 = 3.3334
	// 15*30/90 = 5; 30 - 25 = 5
	assert.Equal(int64(11e8), trades[4].LastPx)
	assert.Equal(int64(4.9999e8), trades[4].LastQty)
	assert.Equal(int64(11e8), trades[5].LastPx)
	assert.Equal(int64(1.6667e8), trades[5].LastQty)
	assert.Equal(int64(11e8), trades[6].LastPx)
	assert.Equal(int64(3.3334e8), trades[6].LastQty)
	assert.Equal(int64(11e8), trades[7].LastPx)
	assert.Equal(int64(5e8), trades[7].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100090e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(98649.5050e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(360e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99910e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100989.5050e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

func Test_Maker_Buy_2a(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 10
	   sum    sell    price    buy    sum    exec    imbal
	   75             10       75(m)  75     75      0
	   75             9        25     100    75      25
	   75             8               100    75      25
	   75     75      7               100    75      25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 75e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99025e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(975e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(75e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int64(25e8), trades[1].LastQty)
	assert.Equal(int64(25e8), trades[2].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100075e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99024.6250e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100749.6250e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

func Test_Maker_Buy_2b(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 10
	   sum    sell    price    buy    sum    exec    imbal
	   75             10       75(m)  75     75      0
	   75             9        25     100    75      25
	   75             8               100    75      25
	   75     75      7               100    75      25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 75e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99525e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(475e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(75e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99750e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(250e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99750e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(250e8), GetLocked(ctx, addr3, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int64(25e8), trades[1].LastQty)
	assert.Equal(int64(25e8), trades[2].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100025e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99524.8750e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100749.6250e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100025e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99749.8750e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100025e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99749.8750e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
}

func Test_Maker_Buy_2c(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 10
	   sum    sell    price    buy    sum    exec    imbal
	   75             10       75(m)  75     75      0
	   75             9        25     100    75      25
	   75             8               100    75      25
	   75     75      7               100    75      25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99525e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(475e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(75e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99750e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(250e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99750e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(250e8), GetLocked(ctx, addr3, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int64(25e8), trades[1].LastQty)
	assert.Equal(int64(25e8), trades[2].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100025e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99524.8750e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99925e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100749.6250e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100025e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99749.8750e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100025e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99749.8750e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
}

func Test_Maker_Buy_2d(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 9
	   sum    sell    price    buy    sum    exec    imbal
	   93             10       18     18     18
	   93             9        75(m)  93     93      0
	   93             8        25     118    93      25
	   93             7               118    93      25
	   93     93      6               118    93      25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 18e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(93e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99395e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(605e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99907e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(93e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99775e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99775e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr3, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int64(6e8), trades[1].LastQty)
	assert.Equal(int64(19e8), trades[2].LastQty)
	assert.Equal(int64(12e8), trades[3].LastQty)
	assert.Equal(int64(13e8), trades[4].LastQty)
	assert.Equal(int64(18e8), trades[5].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100043e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99412.8065e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99907e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100836.5815e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100025e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99774.8875e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100025e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99774.8875e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BNB"))
}

func Test_Maker_Buy_2e(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 9
	   sum    sell    price    buy    sum    exec    imbal
	   60             10       25     25     25
	   60             9        75(m)  100    60      40
	   60             8        25     125    60      65
	   60             7               125    60      65
	   60     60      6               125    60      65
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(60e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99325e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(675e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99940e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99775e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99775e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr3, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	// 60-25 = 35e8, it will be split by 3 users
	// 35/3 = 11.6666e8; given the lot size is 1e4
	// 35-11.6666*3 = 0.0002e8; so it's 2 lots
	// after the split:
	// user a: 11.6666+0.0001 = 11.6667
	// user b: 11.6666+0.0001 = 11.6667
	// user c: 11.6666
	assert.Equal(int64(11.6667e8), trades[0].LastQty)
	assert.Equal(int64(8.3333e8), trades[1].LastQty)
	assert.Equal(int64(3.3334e8), trades[2].LastQty)
	assert.Equal(int64(11.6666e8), trades[3].LastQty)
	assert.Equal(int64(5e8), trades[4].LastQty)
	assert.Equal(int64(20e8), trades[5].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100036.6667e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99349.83499985e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(319.9997e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99940e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100539.7300e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100011.6667e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99774.94749985e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(119.9997e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100011.6666e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99774.94750030e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(120.0006e8), GetLocked(ctx, addr3, "BNB"))
}

func Test_Maker_Buy_2f(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 9
	   sum    sell    price    buy    sum    exec    imbal
	   60             10       25     25     25
	   60             9        75(m)  100    60      40
	   60             8        25     125    60      65
	   60             7               125    60      65
	   60     60      6               125    60      65
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(60e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99325e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(675e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99940e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(60e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99775e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99775e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr3, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(20e8), trades[0].LastQty)
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int64(10e8), trades[2].LastQty)
	assert.Equal(int64(5e8), trades[3].LastQty)
	assert.Equal(int64(20e8), trades[4].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100050e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99349.7750e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99940e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100539.73e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100010e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99774.9550e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(135e8), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(99775e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr3, "BNB"))
}

func Test_Maker_Buy_3a(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 10
	   sum    sell    price    buy    sum    exec    imbal
	   75             10       75(m)  75     75      0
	   75             9        25     100    75      25
	   75             8               100    75      25
	   75     75      7               100    75      25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 30e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 35e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 11e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 13e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 51e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99475e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(525e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99989e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(11e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99987e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(13e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99949e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(51e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99650e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(350e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99900e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(35e8), trades[0].LastQty)
	assert.Equal(int64(16e8), trades[1].LastQty)
	assert.Equal(int64(13e8), trades[2].LastQty)
	assert.Equal(int64(1e8), trades[3].LastQty)
	assert.Equal(int64(10e8), trades[4].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100030e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99474.85e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99989e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100109.9450e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99987e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100129.9350e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99949e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100509.7450e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100035e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99649.8250e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100010e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99899.9500e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

func Test_Maker_Buy_3b(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 10
	   sum    sell    price    buy    sum    exec    imbal
	   75             10       75(m)  75     75      0
	   75             9        25     100    75      25
	   75             8               100    75      25
	   75     75      7               100    75      25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 30e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 35e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 11e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 13e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 51e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(75e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99475e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(525e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99989e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(11e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99987e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(13e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99949e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(51e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99650e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(350e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99900e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(10e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(35e8), trades[0].LastQty)
	assert.Equal(int64(16e8), trades[1].LastQty)
	assert.Equal(int64(13e8), trades[2].LastQty)
	assert.Equal(int64(1e8), trades[3].LastQty)
	assert.Equal(int64(10e8), trades[4].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100030e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99474.85e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99989e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100109.9450e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99987e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100129.9350e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99949e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100509.7450e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100035e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99649.8250e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100010e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99899.9500e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

// maker order limit price > concluded price;
func Test_Maker_Buy_3c(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 9
	   sum    sell    price    buy    sum    exec    imbal
	   93             10       75(m)  75     93      -18
	   93             9        25     100    93      7
	   93             8               100    93      7
	   93     93      7               100    93      7
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 21e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 10e8, 24e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 10e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 7e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 7e8, 31e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(93e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99565e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(435e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99969e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(31e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99969e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(31e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99969e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(31e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99760e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(240e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99700e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(300e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(8, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(10e8), trades[0].LastPx)
	assert.Equal(int64(25e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastPx)
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int64(10e8), trades[2].LastPx)
	assert.Equal(int64(20e8), trades[2].LastQty)
	assert.Equal(int64(10e8), trades[3].LastPx)
	assert.Equal(int64(4e8), trades[3].LastQty)
	assert.Equal(int64(10e8), trades[3].LastPx)
	assert.Equal(int64(21e8), trades[4].LastQty)
	assert.Equal(int64(9e8), trades[5].LastPx)
	assert.Equal(int64(6e8), trades[5].LastQty)
	assert.Equal(int64(9e8), trades[6].LastPx)
	assert.Equal(int64(6e8), trades[6].LastQty)
	assert.Equal(int64(9e8), trades[7].LastPx)
	assert.Equal(int64(6e8), trades[7].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100030e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99474.85e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(225e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99989e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100109.9450e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99987e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100129.9350e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99949e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100509.7450e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100035e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99649.8250e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100010e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99899.9500e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr5, "BNB"))
}

func Test_Maker_Buy_3d(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 9
	   sum    sell    price    buy    sum    exec    imbal
	   60             10       25     25     25
	   60             9        75(m)  100    60      40
	   60             8        25     125    60      65
	   60             7               125    60      65
	   60     60      6               125    60      65
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()
	addr4 := accs[4].GetAddress()
	addr5 := accs[5].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 21e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr4, 0, ctx)
	msg = order.NewNewOrderMsg(addr4, oid, 1, "BTC-000_BNB", 9e8, 24e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr5, 0, ctx)
	msg = order.NewNewOrderMsg(addr5, oid, 1, "BTC-000_BNB", 9e8, 30e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(utils.Fixed8(75e8), buys[0].qty)
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 11e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 2, "BTC-000_BNB", 6e8, 27e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr3, 0, ctx)
	msg = order.NewNewOrderMsg(addr3, oid, 2, "BTC-000_BNB", 6e8, 22e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(60e8), sells[0].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99361e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(639e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99989e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(11e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99973e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(27e8), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99978e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(22e8), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99784e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(216e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99730e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(270e8), GetLocked(ctx, addr5, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(5, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(21e8), trades[0].LastQty)
	assert.Equal(int64(6e8), trades[1].LastQty)
	assert.Equal(int64(8e8), trades[2].LastQty)
	assert.Equal(int64(14e8), trades[3].LastQty)
	assert.Equal(int64(11e8), trades[4].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100046e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99385.7930e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(200e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99989e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100098.9505e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99973e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(100242.8785e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99978e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100197.9010e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100014e8), GetAvail(ctx, addr4, "BTC-000"))
	assert.Equal(int64(99783.9370e8), GetAvail(ctx, addr4, "BNB"))
	assert.Equal(int64(90e8), GetLocked(ctx, addr4, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr5, "BTC-000"))
	assert.Equal(int64(99730e8), GetAvail(ctx, addr5, "BNB"))
	assert.Equal(int64(270e8), GetLocked(ctx, addr5, "BNB"))
}

func Test_Maker_Buy_4(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   59     10      10       5(m)   5      5
	   49     10      9        5      10     10
	   39     9       8        5      15     15
	   32     7       7        20(m)  35     32
	   25             6        5(m)   40     25
	   25     25      5               40     25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 5e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 4, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 9e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 7e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 4, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 5e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(5, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99695e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(305e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99939e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(61e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	// 5*25/32 = 3.9062; + 0.0001 = 3.9063
	// 5*7/32 = 1.0937;
	assert.Equal(int64(10e8), trades[0].LastPx)
	assert.Equal(int64(3.9063e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastPx)
	assert.Equal(int64(1.0937e8), trades[1].LastQty)
	assert.Equal(int64(7e8), trades[2].LastPx)
	assert.Equal(int64(17e8), trades[2].LastQty)
	assert.Equal(int64(7e8), trades[3].LastPx)
	assert.Equal(int64(4.0937e8), trades[3].LastQty)
	assert.Equal(int64(7e8), trades[4].LastPx)
	assert.Equal(int64(0.9063e8), trades[4].LastQty)
	assert.Equal(int64(7e8), trades[5].LastPx)
	assert.Equal(int64(5e8), trades[5].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99570.872e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(173e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99960e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(40e8), GetLocked(ctx, addr1, "BTC-000"))
}

func Test_Maker_Buy_5(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 7
	   sum    sell    price    buy    sum    exec    imbal
	   59     10      10       5(m)   5      5
	   49     10      9        5      10     10
	   39     9       8        5      15     15
	   32     7       7        20(m)  35     32
	   25             6        5(m)   40     25
	   25     25      5               40     25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 5e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 20e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 6e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 4, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 10e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 9e8, 10e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 2, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 9e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 3, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 7e8, 7e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 4, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 5e8, 25e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(5, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99695e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(305e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99939e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(61e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(7e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	// 5*25/32 = 3.9062; + 0.0001 = 3.9063
	// 5*7/32 = 1.0937;
	assert.Equal(int64(10e8), trades[0].LastPx)
	assert.Equal(int64(3.9063e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastPx)
	assert.Equal(int64(1.0937e8), trades[1].LastQty)
	assert.Equal(int64(7e8), trades[2].LastPx)
	assert.Equal(int64(17e8), trades[2].LastQty)
	assert.Equal(int64(7e8), trades[3].LastPx)
	assert.Equal(int64(4.0937e8), trades[3].LastQty)
	assert.Equal(int64(7e8), trades[4].LastPx)
	assert.Equal(int64(0.9063e8), trades[4].LastQty)
	assert.Equal(int64(7e8), trades[5].LastPx)
	assert.Equal(int64(5e8), trades[5].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(3, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99570.872e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(173e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99960e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(40e8), GetLocked(ctx, addr1, "BTC-000"))
}

func Test_Maker_Buy_6(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   32             10       5      5      5
	   32             9        6      11     11
	   32     7       8        20(m)  31     31
	   25             7        5(m)   36     25
	   25     25      6               36     25
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 20e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 6e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 7e8)
	msg.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 25e8)
	msg.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99701e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(299e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99968e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(32e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(20e8), trades[0].LastQty)
	// 5 before 6, as for the price, 10 is better than 9
	assert.Equal(int64(5e8), trades[1].LastQty)
	assert.Equal(int64(6e8), trades[2].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100031e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99716.8760e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(35e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99969e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100247.8760e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

func Test_Maker_Buy_7(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   2              10       5(m)   5      5
	   22             9        6(m)   11     11
	   22     7       8        20(m)  31     22
	   15             7        5      36     15
	   15     15      6               36     15
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 20e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 6e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContext(addr, ctx, 3)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(3, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 4)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 8e8, 7e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 1, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 15e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(2, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99701e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(299e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99978e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(22e8), GetLocked(ctx, addr1, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(6, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	// 5*15/22 = 3.4090; + 0.0001 = 3.4091
	// 5*7/22 = 1.5909
	assert.Equal(int64(10e8), trades[0].LastPx)
	assert.Equal(int64(3.4091e8), trades[0].LastQty)
	assert.Equal(int64(10e8), trades[1].LastPx)
	assert.Equal(int64(1.5909e8), trades[1].LastQty)
	// 6*15/22 = 4.0909; + 0.0001 = 4.0910
	// 6*7/22 = 1.9090;
	assert.Equal(int64(9e8), trades[2].LastPx)
	assert.Equal(int64(4.0910e8), trades[2].LastQty)
	assert.Equal(int64(9e8), trades[3].LastPx)
	assert.Equal(int64(1.9090e8), trades[3].LastQty)
	// 11*15/22 = 7.5; 15-3.4091-4.0910=7.4999
	// 11*7/22 = 3.5; 7-1.5909-1.9090=3.5001
	assert.Equal(int64(8e8), trades[4].LastPx)
	assert.Equal(int64(7.4999e8), trades[4].LastQty)
	assert.Equal(int64(8e8), trades[5].LastPx)
	assert.Equal(int64(3.5001e8), trades[5].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100022e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99751.8760e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99964e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100247.8760e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(14e8), GetLocked(ctx, addr1, "BTC-000"))
}

func Test_Maker_Buy_8(t *testing.T) {
	assert := assert.New(t)

	/* concluded price = 8
	   sum    sell    price    buy    sum    exec    imbal
	   29             10       5      5      5
	   29             9        5(m)   10     10
	   29             8        20     30     29      1
	   29             7        5      35     29      6
	   29     29      6               35     29      6
	*/

	addr, ctx, accs := SetupTest()

	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	addr2 := accs[2].GetAddress()

	ctx = UpdateContext(addr, ctx, 1)

	oid := GetOrderId(addr0, 0, ctx)
	msg := order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 9e8, 5e8)
	_, err := testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	ctx = UpdateContext(addr, ctx, 2)

	oid = GetOrderId(addr0, 1, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 10e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 2, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 8e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr0, 3, ctx)
	msg = order.NewNewOrderMsg(addr0, oid, 1, "BTC-000_BNB", 7e8, 5e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr2, 0, ctx)
	msg = order.NewNewOrderMsg(addr2, oid, 1, "BTC-000_BNB", 8e8, 15e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	oid = GetOrderId(addr1, 0, ctx)
	msg = order.NewNewOrderMsg(addr1, oid, 2, "BTC-000_BNB", 6e8, 29e8)
	_, err = testClient.DeliverTxSync(msg, testApp.Codec)
	assert.NoError(err)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(4, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(utils.Fixed8(20e8), buys[2].qty)

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99830e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(170e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99971e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(29e8), GetLocked(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99880e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(120e8), GetLocked(ctx, addr2, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(8e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d\n", i, trade.LastPx, trade.LastQty)
	}
	assert.Equal(int64(9e8), trades[0].LastPx)
	assert.Equal(int64(5e8), trades[0].LastQty)
	assert.Equal(int64(8e8), trades[1].LastPx)
	assert.Equal(int64(5e8), trades[1].LastQty)
	// 19/20 = 0.95
	// 15*0.95 = 14.25
	// 5*0.95 = 4.75
	assert.Equal(int64(8e8), trades[2].LastPx)
	assert.Equal(int64(14.25e8), trades[2].LastQty)
	assert.Equal(int64(8e8), trades[3].LastPx)
	assert.Equal(int64(4.75e8), trades[3].LastQty)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100022e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99751.8760e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99964e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100247.8760e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(14e8), GetLocked(ctx, addr1, "BTC-000"))
}