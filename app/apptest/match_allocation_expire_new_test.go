package apptest

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/order"
)

/*
test #1: all orders on one side expire, either buy or sell
*/
func Test_Expire_1_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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

	// testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(5, len(transfers))
	for i := 0; i < len(transfers); i++ {
		assert.Equal(int64(2e4), transfers[i].Fee.Tokens[0].Amount)
	}

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

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers = make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(5, len(transfers))
	for i := 0; i < len(transfers); i++ {
		assert.Equal(int64(2e4), transfers[i].Fee.Tokens[0].Amount)
	}

	_, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2a: the first 5 levels expire, for buy
*/
func Test_Expire_2a_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(6, len(buys))
	assert.Equal(0, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(5, len(transfers))
	for i := 0; i < len(transfers); i++ {
		assert.Equal(int64(2e4), transfers[i].Fee.Tokens[0].Amount)
	}

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99899.9990e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2aa: buy side, with the new match engine, match is taken place in breath block, and is prior to expire
*/
func Test_Expire_2aa_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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
	msgB := order.NewNewOrderMsg(addr0, oidB, 1, "BTC-000_BNB", 1e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 1e8, 10e8)
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(5, len(buys))
	assert.Equal(utils.Fixed8(11e8), buys[4].qty)
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	/* concluded price = 3
	   sum    sell    price    buy      sum    exec    imbal
	   10             5        5        5      5
	   10	     	  4        4        9      9
	   10             3	       3        12     10      2
	   10	  	      2	       2        14     10      4
	   10	  10      1        11(10,1) 25     10      14
	*/

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(3e8), lastPx)
	assert.Equal(3, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(5e8), trades[0].LastPx)
	assert.Equal(int64(5e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.SellTaker), trades[0].TickType)
	assert.Equal(int64(0.0125e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0125e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(4e8), trades[1].LastPx)
	assert.Equal(int64(4e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.SellTaker), trades[0].TickType)
	assert.Equal(int64(0.0080e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0080e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(3e8), trades[2].LastPx)
	assert.Equal(int64(1e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.SellTaker), trades[0].TickType)
	assert.Equal(int64(0.0015e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0015e8), trades[2].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100010e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99945.9776e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100043.9780e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2b: the first 5 levels expire, for sell
*/
func Test_Expire_2b_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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

	oidS := GetOrderId(addr1, 5, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 11e8, 10e8)
	_, err := testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(6, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(5, len(transfers))
	for i := 0; i < len(transfers); i++ {
		assert.Equal(int64(2e4), transfers[i].Fee.Tokens[0].Amount)
	}

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99990e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #2bb: sell side, with the new match engine, match is taken place in breath block, and is prior to expire
*/
func Test_Expire_2bb_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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

	/* concluded price = 4
		   sum    sell    price    buy    sum    exec    imbal
		   25     10      11
	       15             10       10     10     10      -5
		   15     5       5               10     10      -5
		   10	  4  	  4               10     10      0
		   6      3       3	              10     6
		   3	  2  	  2	              10     3
		   1	  1       1               10     1
	*/

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(4e8), lastPx)
	assert.Equal(4, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(1e8), trades[0].LastPx)
	assert.Equal(int64(1e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0005e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0005e8), trades[0].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(2e8), trades[1].LastPx)
	assert.Equal(int64(2e8), trades[1].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0020e8), trades[1].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0020e8), trades[1].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(3e8), trades[2].LastPx)
	assert.Equal(int64(3e8), trades[2].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0045e8), trades[2].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0045e8), trades[2].SellerFee.Tokens[0].Amount)
	assert.Equal(int64(4e8), trades[3].LastPx)
	assert.Equal(int64(4e8), trades[3].LastQty)
	assert.Equal(int8(matcheng.BuyTaker), trades[0].TickType)
	assert.Equal(int64(0.0080e8), trades[3].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0080e8), trades[3].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(100010e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99969.9850e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99980e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100029.9848e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(10e8), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #3: orders in the middle levels expire
*/
func Test_Expire_3_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(10, len(transfers))
	for i := 0; i < len(transfers); i++ {
		assert.Equal(int64(2e4), transfers[i].Fee.Tokens[0].Amount)
	}

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
func Test_Expire_4a_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(3, len(transfers))

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
func Test_Expire_4b_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
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

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(3, len(transfers))

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
test #5a: IOC orders, either buy or sell sent in breath block should not expire
note that with the new match engine, it is no longer valid, i.e. IOC orders will be handled as in normal block
no filled and expire
*/
func Test_Expire_5a_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextB(addr, ctx, 1, tNow)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 1e8, 1e8)
	msgB1.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 2e8, 2e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #5b: IOC orders, either buy or sell sent in breath block should not expire
note that with the new match engine, it is no longer valid, i.e. IOC orders will be handled as in normal block
partial filled and expire
*/
func Test_Expire_5b_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new()
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextB(addr, ctx, 1, tNow)

	/* concluded price = 2
	   sum    sell    price    buy    sum    exec    imbal
	   2      2       2        1      1      1
	   0		      1        1      2	     0
	*/

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_BNB", 1e8, 1e8)
	msgB1.TimeInForce = 3
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidB2 := GetOrderId(addr0, 1, ctx)
	msgB2 := order.NewNewOrderMsg(addr0, oidB2, 1, "BTC-000_BNB", 2e8, 1e8)
	msgB2.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidS := GetOrderId(addr1, 0, ctx)
	msgS := order.NewNewOrderMsg(addr1, oidS, 2, "BTC-000_BNB", 2e8, 2e8)
	msgS.TimeInForce = 3
	_, err = testClient.DeliverTxSync(msgS, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_BNB")
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(2e8), lastPx)
	assert.Equal(1, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(1e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.SellSurplus), trades[0].TickType)
	assert.Equal(int64(0.0010e8), trades[0].BuyerFee.Tokens[0].Amount)
	assert.Equal(int64(0.0010e8), trades[0].SellerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(100001e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(99997.9989e8), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(100001.9990e8), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr1, "BTC-000"))
}

/*
test #6: expire fee is larger than the bnb balance, the bnb balance becomes 0
*/
func Test_Expire_6_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new(1e12)
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

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(1, len(transfers))
	for i := 0; i < len(transfers); i++ {
		assert.Equal(int64(1e2), transfers[i].Fee.Tokens[0].Amount)
	}

	buys, _ = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))

	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr0, "BNB"))
}

/*
test #7: no bnb balance, expire fee is charged in the balance of the opposite token
without bnb trade, expire fee of the opposite token is calculated using bnb init price
*/
func Test_Expire_7_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new(10e8, 15e8, 5e8)
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	ResetAccount(ctx, addr0, 0, 100000e8, 100000e8)
	ResetAccount(ctx, addr1, 0, 100000e8, 100000e8)

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextC(addr, ctx, 1)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_ETH-000", 3e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidS1 := GetOrderId(addr1, 0, ctx)
	msgS1 := order.NewNewOrderMsg(addr1, oidS1, 2, "BTC-000_ETH-000", 6e8, 15e8)
	_, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_ETH-000")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(99970e8), GetAvail(ctx, addr0, "ETH-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "ETH-000"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr1, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(2, len(transfers))
	for _, transfer := range transfers {
		fmt.Println(transfer.Oid, transfer.Fee.Tokens[0].Denom)
		if transfer.Fee.Tokens[0].Denom == "ETH-000" {
			// eth expire fee: 0.001 bnb, 0.001/15 = 0.00006666 eth
			assert.Equal(int64(0.00006666e8), transfer.Fee.Tokens[0].Amount)
		}
		if transfer.Fee.Tokens[0].Denom == "BTC-000" {
			// btc expire fee: 0.001 bnb, 0.001/10 = 0.0001 btc
			assert.Equal(int64(0.0001e8), transfer.Fee.Tokens[0].Amount)
		}
	}

	buys, sells = GetOrderBook("BTC-000_ETH-000")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(99999.99993334e8), GetAvail(ctx, addr0, "ETH-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "ETH-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr1, "BNB"))
}

/*
test #7: small bnb balance, expire fee is charged in the balance of the opposite token
without bnb trade, expire fee of the opposite token is calculated using bnb init price
transfers order for the same user: BNB -> BTC -> ETH
*/
func Test_Expire_8_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new(10e8, 15e8, 5e8)
	addr2 := accs[2].GetAddress()
	ResetAccount(ctx, addr2, 0.0002e8, 100000e8, 100000e8)

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextC(addr, ctx, 1)

	oidB2 := GetOrderId(addr2, 0, ctx)
	msgB2 := order.NewNewOrderMsg(addr2, oidB2, 1, "BTC-000_ETH-000", 4e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidS2 := GetOrderId(addr2, 1, ctx)
	msgS2 := order.NewNewOrderMsg(addr2, oidS2, 2, "BTC-000_ETH-000", 7e8, 15e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	assert.NoError(err)

	oidS3 := GetOrderId(addr2, 2, ctx)
	msgS3 := order.NewNewOrderMsg(addr2, oidS3, 2, "BTC-000_BNB", 9e8, 15e8)
	_, err = testClient.DeliverTxSync(msgS3, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_ETH-000")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(99960e8), GetAvail(ctx, addr2, "ETH-000"))
	assert.Equal(int64(99970e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0.0002e8), GetAvail(ctx, addr2, "BNB"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	//testClient.cl.EndBlockSync(abci.RequestEndBlock{})
	transfers := make([]*order.Transfer, 0)
	testApp.DexKeeper.ExpireOrders(ctx, ctx.BlockHeader().Time, func(transfer order.Transfer) {
		transfers = append(transfers, &transfer)
	})
	assert.Equal(3, len(transfers))
	assert.Equal("BNB", transfers[0].Fee.Tokens[0].Denom)
	assert.Equal(int64(0.0002e8), transfers[0].Fee.Tokens[0].Amount)
	assert.Equal("BTC-000", transfers[1].Fee.Tokens[0].Denom)
	assert.Equal(int64(0.0001e8), transfers[1].Fee.Tokens[0].Amount)
	assert.Equal("ETH-000", transfers[2].Fee.Tokens[0].Denom)
	assert.Equal(int64(0.00006666e8), transfers[2].Fee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_ETH-000")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	assert.Equal(int64(99999.99993334e8), GetAvail(ctx, addr2, "ETH-000"))
	assert.Equal(int64(99999.9999e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(0e8), GetAvail(ctx, addr2, "BNB"))
}

/*
test #7: no bnb balance, expire fee is charged in the balance of the opposite token
with bnb trade, expire fee of the opposite token is calculated using latest bnb price
*/
func Test_Expire_9_new(t *testing.T) {
	assert := assert.New(t)

	addr, ctx, accs := SetupTest_new(10e8, 15e8, 5e8)
	addr0 := accs[0].GetAddress()
	addr1 := accs[1].GetAddress()
	ResetAccount(ctx, addr0, 0, 100000e8, 100000e8)
	ResetAccount(ctx, addr1, 0, 100000e8, 100000e8)
	addr2 := accs[2].GetAddress()
	addr3 := accs[3].GetAddress()

	tNow := time.Now()

	testApp.SetCheckState(abci.Header{Time: tNow.AddDate(0, 0, -1)})

	ctx = UpdateContextC(addr, ctx, 1)

	oidB1 := GetOrderId(addr0, 0, ctx)
	msgB1 := order.NewNewOrderMsg(addr0, oidB1, 1, "BTC-000_ETH-000", 3e8, 10e8)
	_, err := testClient.DeliverTxSync(msgB1, testApp.Codec)
	assert.NoError(err)

	oidB2 := GetOrderId(addr2, 0, ctx)
	msgB2 := order.NewNewOrderMsg(addr2, oidB2, 1, "BTC-000_BNB", 9e8, 1e8)
	_, err = testClient.DeliverTxSync(msgB2, testApp.Codec)
	assert.NoError(err)

	oidS1 := GetOrderId(addr1, 0, ctx)
	msgS1 := order.NewNewOrderMsg(addr1, oidS1, 2, "BTC-000_ETH-000", 6e8, 15e8)
	_, err = testClient.DeliverTxSync(msgS1, testApp.Codec)
	assert.NoError(err)

	oidS2 := GetOrderId(addr3, 0, ctx)
	msgS2 := order.NewNewOrderMsg(addr3, oidS2, 2, "BTC-000_BNB", 9e8, 1e8)
	_, err = testClient.DeliverTxSync(msgS2, testApp.Codec)
	assert.NoError(err)

	buys, sells := GetOrderBook("BTC-000_ETH-000")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	buys, sells = GetOrderBook("BTC-000_BNB")
	assert.Equal(1, len(buys))
	assert.Equal(1, len(sells))

	assert.Equal(int64(99970e8), GetAvail(ctx, addr0, "ETH-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "ETH-000"))
	assert.Equal(int64(99985e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(1e8), GetLocked(ctx, addr3, "BTC-000"))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 2, tNow)

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	ctx = UpdateContextB(addr, ctx, 3, tNow.AddDate(0, 0, 3))

	testClient.cl.EndBlockSync(abci.RequestEndBlock{})

	trades, lastPx := testApp.DexKeeper.GetLastTradesForPair("BTC-000_BNB")
	assert.Equal(int64(9e8), lastPx)
	assert.Equal(1, len(trades))
	for i, trade := range trades {
		fmt.Printf("#%d: p: %d; q: %d; s: %d\n",
			i, trade.LastPx, trade.LastQty, trade.TickType)
	}
	assert.Equal(int64(1e8), trades[0].LastQty)
	assert.Equal(int8(matcheng.Neutral), trades[0].TickType)
	assert.Equal(int64(0.0045e8), trades[0].BuyerFee.Tokens[0].Amount)

	buys, sells = GetOrderBook("BTC-000_ETH-000")
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))

	// since there is a trade, BTC-000_BNB price is no longer 10, but 9
	// btc expire fee: 0.001 bnb, 0.001/9 = 0.00011111 btc

	assert.Equal(int64(99999.99993334e8), GetAvail(ctx, addr0, "ETH-000"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr0, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr0, "BNB"))
	assert.Equal(int64(100000e8), GetAvail(ctx, addr1, "ETH-000"))
	assert.Equal(int64(99999.99988889e8), GetAvail(ctx, addr1, "BTC-000"))
	assert.Equal(int64(0), GetAvail(ctx, addr1, "BNB"))
	assert.Equal(int64(100001e8), GetAvail(ctx, addr2, "BTC-000"))
	assert.Equal(int64(99990.9955e8), GetAvail(ctx, addr2, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr2, "BNB"))
	assert.Equal(int64(99999e8), GetAvail(ctx, addr3, "BTC-000"))
	assert.Equal(int64(100008.9955e8), GetAvail(ctx, addr3, "BNB"))
	assert.Equal(int64(0), GetLocked(ctx, addr3, "BTC-000"))
}
