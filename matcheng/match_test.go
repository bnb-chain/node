package matcheng

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_sumOrders(t *testing.T) {
	assert := assert.New(t)
	orders := []OrderPart{OrderPart{"1", 100, 260, 0, 0}, OrderPart{"1", 100, 250, 0, 0}, OrderPart{"1", 100, 501, 0, 0}}
	assert.Equal(int64(1011), sumOrdersTotalLeft(orders, true))
	orders[0].qty = 10
	orders[1].cumQty = 250
	assert.Equal(int64(1011), sumOrdersTotalLeft(orders, false))
	orders = []OrderPart{}
	assert.Equal(int64(0), sumOrdersTotalLeft(orders, true))
	orders = []OrderPart{OrderPart{"1", 100, 260, 0, 0}}
	assert.Equal(int64(260), sumOrdersTotalLeft(orders, true))
	assert.Equal(int64(0), sumOrdersTotalLeft(nil, true))
}

func Test_prepareMatch(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{Price: 1021, BuyOrders: []OrderPart{OrderPart{"1.1", 100, 1500, 0, 0}, OrderPart{"1.2", 102, 1500, 0, 0}}},
		OverLappedLevel{Price: 1001, BuyOrders: []OrderPart{OrderPart{"2.1", 100, 1000, 0, 0}}},
		OverLappedLevel{Price: 991, BuyOrders: []OrderPart{OrderPart{"3.1", 100, 2000, 0, 0}}},
		OverLappedLevel{Price: 981,
			SellOrders: []OrderPart{OrderPart{"4.1", 100, 1000, 0, 0}, OrderPart{"4.2", 101, 1000, 0, 0}, OrderPart{"4.3", 101, 500, 0, 0}},
			BuyOrders:  []OrderPart{OrderPart{"4.4", 100, 3000, 0, 0}}},
		OverLappedLevel{Price: 971, SellOrders: []OrderPart{OrderPart{"5.1", 100, 2500, 0, 0}}},
		OverLappedLevel{Price: 961, SellOrders: []OrderPart{OrderPart{"6.1", 101, 10000, 0, 0}}},
	}
	execs := []int64{3000, 4000, 6000, 9000, 9000, 9000}
	surpluses := []int64{-12000, -11000, -9000, -6000, -3500, -1000}
	assert.Equal(6, prepareMatch(&overlap))
	for i, e := range execs {
		assert.Equal(e, overlap[i].AccumulatedExecutions, fmt.Sprintf("overlap number %d", i))
	}
	for i, e := range surpluses {
		assert.Equal(e, overlap[i].BuySellSurplus, fmt.Sprintf("overlap number %d", i))
	}
}

func Test_getPriceCloseToRef(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{Price: 1021, BuyOrders: []OrderPart{OrderPart{"1.1", 100, 1500, 0, 0}, OrderPart{"1.2", 102, 1500, 0, 0}}},
		OverLappedLevel{Price: 1001, BuyOrders: []OrderPart{OrderPart{"2.1", 100, 1000, 0, 0}}},
		OverLappedLevel{Price: 991, BuyOrders: []OrderPart{OrderPart{"3.1", 100, 2000, 0, 0}}},
		OverLappedLevel{Price: 981,
			SellOrders: []OrderPart{OrderPart{"4.1", 100, 1000, 0, 0}, OrderPart{"4.2", 101, 1000, 0, 0}, OrderPart{"4.3", 101, 500, 0, 0}},
			BuyOrders:  []OrderPart{OrderPart{"4.4", 100, 3000, 0, 0}}},
		OverLappedLevel{Price: 971, SellOrders: []OrderPart{OrderPart{"5.1", 100, 2500, 0, 0}}},
		OverLappedLevel{Price: 961, SellOrders: []OrderPart{OrderPart{"6.1", 101, 10000, 0, 0}}},
	}

	p, i := getPriceCloseToRef(overlap, []int{0, 1, 2}, 990)
	assert.Equal(2, i)
	assert.Equal(int64(991), p)
	p, i = getPriceCloseToRef(overlap, []int{0, 1, 2}, 996)
	assert.Equal(1, i)
	assert.Equal(int64(996), p)
	p, i = getPriceCloseToRef(overlap, []int{0, 1, 2}, 1025)
	assert.Equal(0, i)
	assert.Equal(int64(1021), p)

	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 996)
	assert.Equal(0, i)
	assert.Equal(int64(996), p)
	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 991)
	assert.Equal(2, i)
	assert.Equal(int64(991), p)
	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 1025)
	assert.Equal(0, i)
	assert.Equal(int64(1021), p)
	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 1021)
	assert.Equal(0, i)
	assert.Equal(int64(1021), p)
	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 961)
	assert.Equal(5, i)
	assert.Equal(int64(961), p)

}

func Test_calMaxExec(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{AccumulatedExecutions: 5000},
		OverLappedLevel{AccumulatedExecutions: 3000},
		OverLappedLevel{AccumulatedExecutions: 13001},
		OverLappedLevel{AccumulatedExecutions: 13001},
		OverLappedLevel{AccumulatedExecutions: 13000},
		OverLappedLevel{AccumulatedExecutions: 13001},
		OverLappedLevel{AccumulatedExecutions: 11001},
	}
	maxExec := LevelIndex{}
	calMaxExec(&overlap, &maxExec)
	assert.Equal(int64(13001), maxExec.value)
	assert.Equal(3, len(maxExec.index))
	assert.Equal([]int{2, 3, 5}, maxExec.index)

	maxExec.clear()
	overlap2 := overlap[:2]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(int64(5000), maxExec.value)
	assert.Equal(1, len(maxExec.index))
	assert.Equal([]int{0}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[:3]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(int64(13001), maxExec.value)
	assert.Equal(1, len(maxExec.index))
	assert.Equal([]int{2}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[:1]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(int64(5000), maxExec.value)
	assert.Equal(1, len(maxExec.index))
	assert.Equal([]int{0}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[2:4]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(int64(13001), maxExec.value)
	assert.Equal(2, len(maxExec.index))
	assert.Equal([]int{0, 1}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[2:6]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(int64(13001), maxExec.value)
	assert.Equal(3, len(maxExec.index))
	assert.Equal([]int{0, 1, 3}, maxExec.index)
}

func Test_getTradePrice(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{Price: 1101, AccumulatedExecutions: 5000},
		OverLappedLevel{Price: 1091, AccumulatedExecutions: 3000},
		OverLappedLevel{Price: 1081, AccumulatedExecutions: 13001},
		OverLappedLevel{Price: 1071, AccumulatedExecutions: 14001},
		OverLappedLevel{Price: 1061, AccumulatedExecutions: 13000},
		OverLappedLevel{Price: 1051, AccumulatedExecutions: 13001},
		OverLappedLevel{Price: 1041, AccumulatedExecutions: 11001},
	}
	//simple case for exec
	maxExec := LevelIndex{}
	leastSurplus := SurplusIndex{}
	p, i := getTradePrice(&overlap, &maxExec, &leastSurplus, 0, 0.05)
	assert.Equal(int64(1071), p)
	assert.Equal(3, i)
	overlap = []OverLappedLevel{
		OverLappedLevel{Price: 1101, AccumulatedExecutions: 5000, BuySellSurplus: -8000},
		OverLappedLevel{Price: 1091, AccumulatedExecutions: 3000, BuySellSurplus: -7000},
		OverLappedLevel{Price: 1081, AccumulatedExecutions: 13001, BuySellSurplus: -6000},
		OverLappedLevel{Price: 1071, AccumulatedExecutions: 13001, BuySellSurplus: 5000},
		OverLappedLevel{Price: 1061, AccumulatedExecutions: 13000, BuySellSurplus: 8000},
		OverLappedLevel{Price: 1051, AccumulatedExecutions: 13001, BuySellSurplus: 18000},
		OverLappedLevel{Price: 1041, AccumulatedExecutions: 11001, BuySellSurplus: 28000},
	}
	//simple case for surplus
	maxExec.clear()
	leastSurplus.clear()
	p, i = getTradePrice(&overlap, &maxExec, &leastSurplus, 0, 0.05)
	assert.Equal(int64(1071), p)
	assert.Equal(3, i)

	// implement all the example cases on docs
	/* 	1. Choose the largest execution (Step 1)
	-------------------------------------------------------------
	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	300            100      150    150    150          -150
	300            99              150    150          -150
	300    250     98       150    300    300*         0
	50     50      97              300    50           250
	*/
	me := NewMatchEng(100, 1, 0.05)
	book := NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 100, 150)
	book.InsertOrder("2", SELLSIDE, 100, 98, 250)
	book.InsertOrder("3", SELLSIDE, 101, 97, 50)
	book.InsertOrder("4", BUYSIDE, 101, 98, 150)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 1000, 0.05)
	assert.Equal(int64(98), p)
	assert.Equal(1, i)

	/* 	2. Choose the largest execution (Step 1)
	--------------------------------------------------------------
	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	300            100      150    150    150          -150
	300            99       50     200    200          -100
	300            98              200    200          -100
	300    200     97       300    500    300*         200
	100    100     96              500    100          400
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 100, 150)
	book.InsertOrder("2", SELLSIDE, 100, 96, 100)
	book.InsertOrder("3", SELLSIDE, 101, 97, 200)
	book.InsertOrder("4", BUYSIDE, 101, 99, 50)
	book.InsertOrder("5", BUYSIDE, 102, 97, 300)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100, 0.05)
	assert.Equal(int64(97), p)
	assert.Equal(2, i)

	/* 3. the least abs surplus imbalance (Step 2)
	--------------------------------------------------------------
	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	1500           102      300    300    300          -1200
	1500           101             300    300          -1200
	1500           100      100    400    400          -1100
	1500           99       200    600    600          -900
	1500   250     98       300    900    900          -600
	1250   250     97              900    900          -350
	1000   1000    96              900    900          -100*
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 102, 300)
	book.InsertOrder("2", BUYSIDE, 101, 100, 100)
	book.InsertOrder("3", SELLSIDE, 101, 98, 250)
	book.InsertOrder("4", BUYSIDE, 101, 99, 200)
	book.InsertOrder("5", BUYSIDE, 102, 98, 300)
	book.InsertOrder("6", SELLSIDE, 102, 97, 250)
	book.InsertOrder("7", SELLSIDE, 103, 96, 1000)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100, 0.05)
	assert.Equal(int64(96), p)
	assert.Equal(5, i)

	/* 	4. the least abs surplus imbalance (Step 2)
	--------------------------------------------------------------
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

	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 102, 30)
	book.InsertOrder("2", BUYSIDE, 101, 101, 10)
	book.InsertOrder("3", SELLSIDE, 101, 98, 10)
	book.InsertOrder("4", BUYSIDE, 101, 99, 50)
	book.InsertOrder("5", BUYSIDE, 102, 96, 15)
	book.InsertOrder("6", SELLSIDE, 102, 97, 50)
	book.InsertOrder("7", SELLSIDE, 103, 95, 50)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100, 0.05)
	assert.Equal(int64(97), p)
	assert.Equal(4, i)

	/* 	5.1 choose the lowest for all the same value of sell surplus imbalance,
	reference price is 80 and 5% upper limit (Step 3)
	--------------------------------------------------------------
	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	50             102      10     10     10           -40
	50             101             10     10           -40
	50             100             10     10           -40
	50             99              10     10           -40
	50             98              10     10           -40
	50             97       10     20     20           -30
	50             96              20     20           -30
	50     50      95              20     20           -30*
	*/

	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 102, 10)
	book.InsertOrder("2", BUYSIDE, 101, 97, 10)
	book.InsertOrder("3", SELLSIDE, 101, 95, 50)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 80, 0.05)
	assert.Equal(int64(95), p)
	assert.Equal(2, i)

	/*
		5.2 choose the lowest for all the same value of sell surplus imbalance,
		reference price is 100 and 5% upper limit (Step 3)
		--------------------------------------------------------------
		SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
		50             99       10     10     10           -40
		50             98              10     10           -40
		50             97              10     10           -40
		50             96              10     10           -40
		50             95              10     10           -40
		50             94       10     20     20           -30*
		50             93              20     20           -30
		50     50      92              20     20           -30
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", SELLSIDE, 100, 92, 50)
	book.InsertOrder("2", BUYSIDE, 101, 99, 10)
	book.InsertOrder("3", BUYSIDE, 101, 94, 10)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100, 0.05)
	assert.Equal(int64(94), p)
	assert.Equal(1, i)

	/*
		5.3 choose the lowest for all the same value of sell surplus imbalance,
		reference price is 90 and 5% upper limit (Step 3)
		--------------------------------------------------------------
		SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
		50             99       100    100    50           50
		50             98              100    50           50
		50             97              100    50           50
		50             96              100    50           50
		50             95              100    50           50*
		50             94              100    50           50
		50             93              100    50           50
		50     50      92              100    50           50
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", SELLSIDE, 100, 92, 50)
	book.InsertOrder("2", BUYSIDE, 101, 99, 100)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 90, 0.05)
	assert.Equal(int64(95), p)
	assert.Equal(0, i)

	/*
		5.4 choose the lowest for all the same value of sell surplus imbalance,
		reference price is 100 and 5% upper limit (Step 3)
		--------------------------------------------------------------
		SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
		50             101      10     10     10           -40
		50             100             10     10           -40
		50             99              10     10           -40
		50             98              10     10           -40
		50             97              10     10           -40
		50             96       10     20     20           -30
		50             95              20     20           -30*
		50     50      94              20     20           -30
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", SELLSIDE, 100, 94, 50)
	book.InsertOrder("2", BUYSIDE, 101, 96, 10)
	book.InsertOrder("3", BUYSIDE, 101, 101, 10)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100, 0.05)
	assert.Equal(int64(95), p)
	assert.Equal(1, i)
	/*
		6.1 choose the closest to the last trade price 99 (Step 4)
		--------------------------------------------------------------
		SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
		50             100      25     25     25           -25
		50             99              25     25           -25*
		50     25      98              25     25           -25
		25             97       25     50     25           25
		25             96              50     25           25
		25     25      95              50     25           25
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 100, 25)
	book.InsertOrder("2", SELLSIDE, 100, 95, 25)
	book.InsertOrder("2", SELLSIDE, 100, 98, 25)
	book.InsertOrder("3", BUYSIDE, 101, 97, 25)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 99, 0.05)
	assert.Equal(int64(99), p)
	assert.Equal(0, i)
	/*
		6.2 choose the closest to the last trade price 97 (Step 4)
		--------------------------------------------------------------
		SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
		50             100      25     25     25           -25
		50             99              25     25           -25
		50     25      98              25     25           -25
		25             97       25     50     25           25*
		25             96              50     25           25
		25     25      95              50     25           25
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 100, 25)
	book.InsertOrder("2", SELLSIDE, 100, 95, 25)
	book.InsertOrder("2", SELLSIDE, 100, 98, 25)
	book.InsertOrder("3", BUYSIDE, 101, 97, 25)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 97, 0.05)
	assert.Equal(int64(97), p)
	assert.Equal(2, i)
}
func Test_calLeastSurplus(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{AccumulatedExecutions: 5000, BuySellSurplus: -8000},
		OverLappedLevel{AccumulatedExecutions: 3000, BuySellSurplus: -7000},
		OverLappedLevel{AccumulatedExecutions: 13001, BuySellSurplus: -6000},
		OverLappedLevel{AccumulatedExecutions: 13001, BuySellSurplus: -5000},
		OverLappedLevel{AccumulatedExecutions: 13000, BuySellSurplus: 3000},
		OverLappedLevel{AccumulatedExecutions: 13001, BuySellSurplus: 4000},
		OverLappedLevel{AccumulatedExecutions: 13001, BuySellSurplus: -5000},
		OverLappedLevel{AccumulatedExecutions: 12001, BuySellSurplus: 5000},
		OverLappedLevel{AccumulatedExecutions: 13001, BuySellSurplus: 5000},
	}
	me := NewMatchEng(100, 5, 0.05)
	maxExec := me.maxExec
	leastSurplus := me.leastSurplus
	calMaxExec(&overlap, &maxExec)
	calLeastSurplus(&overlap, &maxExec, &leastSurplus)
	assert.Equal([]int{5}, leastSurplus.index)
	assert.Equal(int64(4000), leastSurplus.value)
	assert.Equal([]int64{4000}, leastSurplus.surplus)

	overlap2 := overlap[:4]
	maxExec.clear()
	leastSurplus.clear()
	calMaxExec(&overlap2, &maxExec)
	calLeastSurplus(&overlap2, &maxExec, &leastSurplus)
	assert.Equal([]int{3}, leastSurplus.index)
	assert.Equal(int64(5000), leastSurplus.value)
	assert.Equal([]int64{-5000}, leastSurplus.surplus)

	overlap2 = overlap[6:]
	maxExec.clear()
	leastSurplus.clear()
	calMaxExec(&overlap2, &maxExec)
	calLeastSurplus(&overlap2, &maxExec, &leastSurplus)
	assert.Equal([]int{0, 2}, leastSurplus.index)
	assert.Equal(int64(5000), leastSurplus.value)
	assert.Equal([]int64{-5000, 5000}, leastSurplus.surplus)
}

func TestMatchEng_fillOrders(t *testing.T) {
	assert := assert.New(t)
	me := NewMatchEng(100, 5, 0.05)
	me.lastTradePrice = 999
	me.overLappedLevel = []OverLappedLevel{OverLappedLevel{Price: 1000,
		BuyOrders: []OrderPart{
			OrderPart{"2", 100, 80, 0, 0},
			OrderPart{"1", 100, 70, 0, 0},
			OrderPart{"4", 100, 50, 0, 0},
			OrderPart{"3", 100, 100, 0, 0},
		},
		SellOrders: []OrderPart{
			OrderPart{"9", 100, 60, 0, 0},
			OrderPart{"8", 100, 70, 0, 0},
			OrderPart{"7", 100, 50, 0, 0},
			OrderPart{"6", 100, 100, 0, 0},
		},
	}}
	prepareMatch(&me.overLappedLevel)
	t.Log(me.overLappedLevel)
	assert.Equal(int64(280), me.overLappedLevel[0].AccumulatedExecutions)
	me.fillOrders(0, 0)
	assert.Equal(int64(20), me.overLappedLevel[0].BuyTotal)
	assert.Equal(int64(0), me.overLappedLevel[0].SellTotal)
	t.Log(me.trades)
	assert.Equal([]Trade{
		Trade{"6", 999, 70, "1"},
		Trade{"6", 999, 30, "2"},
		Trade{"7", 999, 50, "2"},
		Trade{"8", 999, 70, "3"},
		Trade{"9", 999, 30, "3"},
		Trade{"9", 999, 30, "4"},
	}, me.trades)

	me.trades = me.trades[:0]
	me.overLappedLevel = []OverLappedLevel{OverLappedLevel{Price: 1000,
		BuyOrders: []OrderPart{
			OrderPart{"2", 100, 80, 0, 0},
			OrderPart{"1", 100, 70, 0, 0},
			OrderPart{"4", 100, 50, 0, 0},
			OrderPart{"3", 100, 100, 0, 0},
		},
		SellOrders: []OrderPart{}},
		OverLappedLevel{Price: 1000,
			BuyOrders: []OrderPart{},
			SellOrders: []OrderPart{
				OrderPart{"9", 100, 60, 0, 0},
				OrderPart{"8", 100, 70, 0, 0},
				OrderPart{"7", 100, 50, 0, 0},
				OrderPart{"6", 100, 100, 0, 0},
			}},
	}
	prepareMatch(&me.overLappedLevel)
	t.Log(me.overLappedLevel)
	assert.Equal(int64(280), me.overLappedLevel[0].AccumulatedExecutions)
	me.fillOrders(0, 1)
	assert.Equal(int64(20), me.overLappedLevel[0].BuyTotal)
	assert.Equal(int64(0), me.overLappedLevel[1].SellTotal)
	t.Log(me.trades)
	assert.Equal([]Trade{
		Trade{"6", 999, 70, "1"},
		Trade{"6", 999, 30, "2"},
		Trade{"7", 999, 50, "2"},
		Trade{"8", 999, 70, "3"},
		Trade{"9", 999, 30, "3"},
		Trade{"9", 999, 30, "4"},
	}, me.trades)
}

func Test_allocateResidual(t *testing.T) {
	assert := assert.New(t)
	orders := []OrderPart{
		OrderPart{"1", 100, 1800, 900, 900},
	}
	var toAlloc int64 = 500
	assert.True(allocateResidual(&toAlloc, orders, 5))
	assert.Equal(int64(500), orders[0].nxtTrade)
	assert.Equal(int64(0), toAlloc)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"3", 100, 600, 0, 600},
		OrderPart{"2", 100, 300, 0, 300},
	}
	toAlloc = 600
	assert.True(allocateResidual(&toAlloc, orders, 5))
	assert.Equal(int64(300), orders[0].nxtTrade)
	assert.Equal(int64(100), orders[1].nxtTrade)
	assert.Equal("2", orders[1].id)
	assert.Equal(int64(200), orders[2].nxtTrade)
	assert.Equal("3", orders[2].id)
	assert.Equal(int64(0), toAlloc)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"3", 100, 600, 0, 600},
		OrderPart{"2", 100, 300, 0, 300},
	}
	toAlloc = 500
	assert.True(allocateResidual(&toAlloc, orders, 5))
	assert.Equal(int64(255), orders[0].nxtTrade)
	assert.Equal(int64(80), orders[1].nxtTrade)
	assert.Equal("2", orders[1].id)
	assert.Equal(int64(165), orders[2].nxtTrade)
	assert.Equal("3", orders[2].id)
	assert.Equal(int64(0), toAlloc)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"3", 100, 600, 0, 600},
		OrderPart{"2", 100, 300, 0, 300},
	}
	toAlloc = 25
	assert.True(allocateResidual(&toAlloc, orders, 5))
	assert.Equal(int64(15), orders[0].nxtTrade)
	assert.Equal(int64(5), orders[1].nxtTrade)
	assert.Equal("2", orders[1].id)
	assert.Equal(int64(5), orders[2].nxtTrade)
	assert.Equal("3", orders[2].id)
	assert.Equal(int64(0), toAlloc)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"3", 100, 600, 0, 600},
		OrderPart{"2", 100, 300, 0, 300},
	}
	toAlloc = 35
	assert.True(allocateResidual(&toAlloc, orders, 5))
	assert.Equal(int64(20), orders[0].nxtTrade)
	assert.Equal(int64(5), orders[1].nxtTrade)
	assert.Equal("2", orders[1].id)
	assert.Equal(int64(10), orders[2].nxtTrade)
	assert.Equal("3", orders[2].id)
	assert.Equal(int64(0), toAlloc)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"3", 100, 900, 0, 900},
		OrderPart{"2", 100, 900, 0, 900},
	}
	toAlloc = 700
	assert.True(allocateResidual(&toAlloc, orders, 5))
	assert.Equal(int64(235), orders[0].nxtTrade)
	assert.Equal(int64(235), orders[1].nxtTrade)
	assert.Equal("2", orders[1].id)
	assert.Equal(int64(230), orders[2].nxtTrade)
	assert.Equal("3", orders[2].id)
	assert.Equal(int64(0), toAlloc)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"2", 100, 900, 0, 900},
		OrderPart{"3", 100, 900, 0, 900},
	}
	toAlloc = 700
	assert.True(allocateResidual(&toAlloc, orders, 5))
	assert.Equal(int64(235), orders[0].nxtTrade)
	assert.Equal(int64(235), orders[1].nxtTrade)
	assert.Equal("2", orders[1].id)
	assert.Equal(int64(230), orders[2].nxtTrade)
	assert.Equal("3", orders[2].id)
	assert.Equal(int64(0), toAlloc)
}

func TestMatchEng_reserveQty(t *testing.T) {
	me := NewMatchEng(100, 5, 0.05)
	assert := assert.New(t)
	orders := []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
	}
	assert.True(me.reserveQty(700, orders))
	assert.Equal(int64(700), orders[0].nxtTrade)
	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"2", 100, 900, 0, 900},
		OrderPart{"3", 100, 900, 0, 900},
	}

	assert.True(me.reserveQty(900, orders))
	assert.Equal(int64(300), orders[0].nxtTrade)
	assert.Equal(int64(300), orders[1].nxtTrade)
	assert.Equal(int64(300), orders[0].nxtTrade)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"2", 100, 900, 0, 900},
		OrderPart{"3", 100, 900, 0, 900},
	}

	assert.True(me.reserveQty(700, orders))
	assert.Equal(int64(235), orders[0].nxtTrade)
	assert.Equal(int64(235), orders[1].nxtTrade)
	assert.Equal(int64(230), orders[2].nxtTrade)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"2", 100, 900, 0, 900},
		OrderPart{"3", 101, 900, 0, 900},
	}

	assert.True(me.reserveQty(700, orders))
	assert.Equal(int64(350), orders[0].nxtTrade)
	assert.Equal(int64(350), orders[1].nxtTrade)
	assert.Equal(int64(0), orders[2].nxtTrade)

	orders = []OrderPart{
		OrderPart{"1", 100, 900, 0, 900},
		OrderPart{"2", 100, 900, 0, 900},
		OrderPart{"3", 101, 900, 0, 900},
		OrderPart{"6", 101, 900, 0, 900},
		OrderPart{"5", 102, 900, 0, 900},
		OrderPart{"4", 102, 900, 0, 900},
		OrderPart{"7", 102, 900, 0, 900},
	}

	assert.True(me.reserveQty(4300, orders))
	assert.Equal(int64(900), orders[0].nxtTrade)
	assert.Equal("1", orders[0].id)
	assert.Equal(int64(900), orders[1].nxtTrade)
	assert.Equal("2", orders[1].id)
	assert.Equal(int64(900), orders[2].nxtTrade)
	assert.Equal("3", orders[2].id)
	assert.Equal(int64(900), orders[3].nxtTrade)
	assert.Equal("6", orders[3].id)
	assert.Equal(int64(235), orders[4].nxtTrade)
	assert.Equal("4", orders[4].id)
	assert.Equal(int64(235), orders[5].nxtTrade)
	assert.Equal("5", orders[5].id)
	assert.Equal(int64(230), orders[6].nxtTrade)
	assert.Equal("7", orders[6].id)
}

func TestMatchEng_Match(t *testing.T) {
	me := NewMatchEng(100, 1, 0.05)
	assert := assert.New(t)
	me.Book = NewOrderBookOnULList(4, 2)
	me.Book.InsertOrder("3", SELLSIDE, 100, 98, 100)
	me.Book.InsertOrder("5", SELLSIDE, 101, 98, 100)
	me.Book.InsertOrder("1", BUYSIDE, 102, 100, 50)
	me.Book.InsertOrder("8", BUYSIDE, 103, 98, 150)
	me.Book.InsertOrder("2", BUYSIDE, 103, 100, 80)
	me.Book.InsertOrder("4", BUYSIDE, 104, 100, 20)
	me.Book.InsertOrder("6", BUYSIDE, 105, 100, 50)
	me.Book.InsertOrder("9", SELLSIDE, 106, 98, 50)
	me.Book.InsertOrder("91", BUYSIDE, 107, 100, 50)
	me.Book.InsertOrder("92", SELLSIDE, 108, 90, 50)

	assert.True(me.Match())
	assert.Equal(3, len(me.overLappedLevel))
	assert.Equal(int64(98), me.lastTradePrice)
	assert.Equal("[{92 98 50 1} {3 98 80 2} {3 98 20 4} {5 98 50 6} {5 98 50 91} {9 98 50 8}]", fmt.Sprint(me.trades))

	me.Book = NewOrderBookOnULList(4, 2)
	me.Book.InsertOrder("3", SELLSIDE, 100, 101, 100)
	me.Book.InsertOrder("5", SELLSIDE, 101, 101, 100)
	me.Book.InsertOrder("1", BUYSIDE, 102, 100, 50)
	me.Book.InsertOrder("8", BUYSIDE, 103, 98, 150)
	me.Book.InsertOrder("2", BUYSIDE, 103, 100, 80)
	me.Book.InsertOrder("4", BUYSIDE, 104, 100, 20)
	me.Book.InsertOrder("6", BUYSIDE, 105, 100, 50)
	me.Book.InsertOrder("9", SELLSIDE, 106, 101, 50)
	me.Book.InsertOrder("91", BUYSIDE, 107, 100, 50)
	me.Book.InsertOrder("92", SELLSIDE, 108, 102, 50)
	assert.True(me.Match())
	assert.Equal(0, len(me.overLappedLevel))
	assert.Equal(0, len(me.trades))

	me.Book = NewOrderBookOnULList(4, 2)
	me.Book.InsertOrder("3", SELLSIDE, 100, 98, 100)
	me.Book.InsertOrder("5", SELLSIDE, 101, 99, 100)
	me.Book.InsertOrder("1", BUYSIDE, 102, 100, 100)
	me.Book.InsertOrder("8", BUYSIDE, 103, 99, 100)

	assert.True(me.Match())
	assert.Equal(3, len(me.overLappedLevel))
	assert.Equal("[{3 99 100 1} {5 99 100 8}]", fmt.Sprint(me.trades))

	me.Book = NewOrderBookOnULList(4, 2)
	me.Book.InsertOrder("3", SELLSIDE, 100, 98, 100)
	me.Book.InsertOrder("5", SELLSIDE, 101, 98, 100)
	me.Book.InsertOrder("1", BUYSIDE, 102, 100, 50)
	me.Book.InsertOrder("8", SELLSIDE, 103, 98, 150)
	me.Book.InsertOrder("2", BUYSIDE, 103, 100, 80)
	me.Book.InsertOrder("4", BUYSIDE, 104, 100, 20)
	me.Book.InsertOrder("6", BUYSIDE, 105, 100, 50)
	me.Book.InsertOrder("9", SELLSIDE, 106, 98, 50)
	me.Book.InsertOrder("91", BUYSIDE, 107, 100, 50)
	me.Book.InsertOrder("92", SELLSIDE, 108, 97, 50)

	assert.True(me.Match())
	assert.Equal(3, len(me.overLappedLevel))
	assert.Equal("[{92 98 50 1} {3 98 80 2} {3 98 20 4} {5 98 50 6} {5 98 50 91}]", fmt.Sprint(me.trades))

	/* 	3. the least abs surplus imbalance (Step 2)
	--------------------------------------------------------------
	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	900            102      250    250    250          -650
	900            101      250    500    500          -400
	900   300      100      1150   1650   900          750*
	600   200      99              1650   900          1050
	400   100      98              1650   900          1250
	900            97              1650   900          1250
	300   300      96              1650   900          1350 */
	me.Book = NewOrderBookOnULList(4, 2)
	me.Book.InsertOrder("3", SELLSIDE, 100, 96, 300)
	me.Book.InsertOrder("5", SELLSIDE, 101, 98, 100)
	me.Book.InsertOrder("1", BUYSIDE, 102, 100, 150)
	me.Book.InsertOrder("8", SELLSIDE, 103, 99, 200)
	me.Book.InsertOrder("31", BUYSIDE, 103, 100, 50)
	me.Book.InsertOrder("2", BUYSIDE, 103, 102, 250)
	me.Book.InsertOrder("4", BUYSIDE, 104, 101, 250)
	me.Book.InsertOrder("6", BUYSIDE, 105, 100, 350)
	me.Book.InsertOrder("9", SELLSIDE, 105, 100, 200)
	me.Book.InsertOrder("91", BUYSIDE, 105, 100, 300)
	me.Book.InsertOrder("92", SELLSIDE, 105, 100, 100)
	me.Book.InsertOrder("93", BUYSIDE, 105, 100, 300)

	assert.True(me.Match())
	t.Log(me.overLappedLevel)
	assert.Equal(6, len(me.overLappedLevel))
	assert.Equal(int64(100), me.lastTradePrice)

	t.Log(me.trades)
}

func TestMatchEng_DropFilledOrder(t *testing.T) {
	me := NewMatchEng(100, 1, 0.05)
	assert := assert.New(t)
	/* 	3. the least abs surplus imbalance (Step 2)
	--------------------------------------------------------------
	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	900            102      250    250    250          -650
	900            101      250    500    500          -400
	900   300      100      1150   1650   900          750*
	600   200      99              1650   900          1050
	400   100      98              1650   900          1250
	900            97              1650   900          1250
	300   300      96              1650   900          1350 */
	book := NewOrderBookOnULList(4, 2)
	me.Book = book
	me.Book.InsertOrder("3", SELLSIDE, 100, 96, 300)
	me.Book.InsertOrder("5", SELLSIDE, 101, 98, 100)
	me.Book.InsertOrder("1", BUYSIDE, 102, 100, 150)
	me.Book.InsertOrder("8", SELLSIDE, 103, 99, 200)
	me.Book.InsertOrder("31", BUYSIDE, 103, 100, 50)
	me.Book.InsertOrder("2", BUYSIDE, 103, 102, 250)
	me.Book.InsertOrder("4", BUYSIDE, 104, 101, 250)
	me.Book.InsertOrder("6", BUYSIDE, 105, 100, 350)
	me.Book.InsertOrder("9", SELLSIDE, 105, 100, 200)
	me.Book.InsertOrder("91", BUYSIDE, 105, 100, 300)
	me.Book.InsertOrder("92", SELLSIDE, 105, 100, 100)
	me.Book.InsertOrder("93", BUYSIDE, 105, 100, 300)

	assert.True(me.Match())
	t.Log(me.overLappedLevel)
	assert.Equal(6, len(me.overLappedLevel))
	assert.Equal(int64(100), me.lastTradePrice)

	t.Log(me.trades)
	assert.Equal(6, me.DropFilledOrder())
	assert.Nil(book.buyQueue.GetPriceLevel(102))
	assert.Nil(book.buyQueue.GetPriceLevel(101))
	assert.Nil(book.sellQueue.GetPriceLevel(100))
	assert.Nil(book.sellQueue.GetPriceLevel(99))
	assert.Nil(book.sellQueue.GetPriceLevel(98))
	assert.Nil(book.sellQueue.GetPriceLevel(96))
	for _, o := range book.buyQueue.GetPriceLevel(100).orders {
		assert.Equal(o.time, uint64(105))
		assert.True(o.cumQty > 0)
	}
}
