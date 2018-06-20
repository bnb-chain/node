package matcheng

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_sumOrders(t *testing.T) {
	assert := assert.New(t)
	orders := []OrderPart{OrderPart{"1", 100, 26.0}, OrderPart{"1", 100, 25.0}, OrderPart{"1", 100, 50.1}}
	assert.Equal(101.1, sumOrders(orders))
	orders = []OrderPart{}
	assert.Equal(0.0, sumOrders(orders))
	orders = []OrderPart{OrderPart{"1", 100, 26.0}}
	assert.Equal(26.0, sumOrders(orders))
	assert.Equal(0.0, sumOrders(nil))
}

func Test_prepareMatch(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{Price: 102.1, BuyOrders: []OrderPart{OrderPart{"1.1", 100, 150.0}, OrderPart{"1.2", 102, 150.0}}},
		OverLappedLevel{Price: 100.1, BuyOrders: []OrderPart{OrderPart{"2.1", 100, 100.0}}},
		OverLappedLevel{Price: 99.1, BuyOrders: []OrderPart{OrderPart{"3.1", 100, 200.0}}},
		OverLappedLevel{Price: 98.1, SellOrders: []OrderPart{OrderPart{"4.1", 100, 100.0}, OrderPart{"4.2", 101, 100.0}, OrderPart{"4.3", 101, 50.0}},
			BuyOrders: []OrderPart{OrderPart{"4.4", 100, 300.0}}},
		OverLappedLevel{Price: 97.1, SellOrders: []OrderPart{OrderPart{"5.1", 100, 250.0}}},
		OverLappedLevel{Price: 96.1, SellOrders: []OrderPart{OrderPart{"6.1", 101, 1000.0}}},
	}
	execs := []float64{300.0, 400.0, 600.0, 900.0, 900.0, 900.0}
	surpluses := []float64{-1200.0, -1100.0, -900.0, -600.0, -350.0, -100.0}
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
		OverLappedLevel{Price: 102.1, BuyOrders: []OrderPart{OrderPart{"1.1", 100, 150.0}, OrderPart{"1.2", 102, 150.0}}},
		OverLappedLevel{Price: 100.1, BuyOrders: []OrderPart{OrderPart{"2.1", 100, 100.0}}},
		OverLappedLevel{Price: 99.1, BuyOrders: []OrderPart{OrderPart{"3.1", 100, 200.0}}},
		OverLappedLevel{Price: 98.1, SellOrders: []OrderPart{OrderPart{"4.1", 100, 100.0}, OrderPart{"4.2", 101, 100.0}, OrderPart{"4.3", 101, 50.0}},
			BuyOrders: []OrderPart{OrderPart{"4.4", 100, 300.0}}},
		OverLappedLevel{Price: 97.1, SellOrders: []OrderPart{OrderPart{"5.1", 100, 250.0}}},
		OverLappedLevel{Price: 96.1, SellOrders: []OrderPart{OrderPart{"6.1", 101, 1000.0}}},
	}

	p, i := getPriceCloseToRef(overlap, []int{0, 1, 2}, 99.0)
	assert.Equal(2, i)
	assert.Equal(99.1, p)
	p, i = getPriceCloseToRef(overlap, []int{0, 1, 2}, 99.6)
	assert.Equal(1, i)
	assert.Equal(100.1, p)
	p, i = getPriceCloseToRef(overlap, []int{0, 1, 2}, 102.5)
	assert.Equal(0, i)
	assert.Equal(102.1, p)

	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 99.6)
	assert.Equal(2, i)
	assert.Equal(99.1, p)
	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 102.5)
	assert.Equal(0, i)
	assert.Equal(102.1, p)
	p, i = getPriceCloseToRef(overlap, []int{0, 2, 5}, 97.5)
	assert.Equal(5, i)
	assert.Equal(96.1, p)
}

func Test_calMaxExec(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{AccumulatedExecutions: 500.0},
		OverLappedLevel{AccumulatedExecutions: 300.0},
		OverLappedLevel{AccumulatedExecutions: 1300.125},
		OverLappedLevel{AccumulatedExecutions: 1300.125},
		OverLappedLevel{AccumulatedExecutions: 1300.0},
		OverLappedLevel{AccumulatedExecutions: 1300.125},
		OverLappedLevel{AccumulatedExecutions: 1100.125},
	}
	maxExec := LevelIndex{}
	calMaxExec(&overlap, &maxExec)
	assert.Equal(1300.125, maxExec.value)
	assert.Equal(3, len(maxExec.index))
	assert.Equal([]int{2, 3, 5}, maxExec.index)

	maxExec.clear()
	overlap2 := overlap[:2]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(500.0, maxExec.value)
	assert.Equal(1, len(maxExec.index))
	assert.Equal([]int{0}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[:3]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(1300.125, maxExec.value)
	assert.Equal(1, len(maxExec.index))
	assert.Equal([]int{2}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[:1]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(500.0, maxExec.value)
	assert.Equal(1, len(maxExec.index))
	assert.Equal([]int{0}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[2:4]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(1300.125, maxExec.value)
	assert.Equal(2, len(maxExec.index))
	assert.Equal([]int{0, 1}, maxExec.index)

	maxExec.clear()
	overlap2 = overlap[2:6]
	calMaxExec(&overlap2, &maxExec)
	assert.Equal(1300.125, maxExec.value)
	assert.Equal(3, len(maxExec.index))
	assert.Equal([]int{0, 1, 3}, maxExec.index)
}

func Test_getTradePrice(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{Price: 110.1, AccumulatedExecutions: 500.0},
		OverLappedLevel{Price: 109.1, AccumulatedExecutions: 300.0},
		OverLappedLevel{Price: 108.1, AccumulatedExecutions: 1300.125},
		OverLappedLevel{Price: 107.1, AccumulatedExecutions: 1400.125},
		OverLappedLevel{Price: 106.1, AccumulatedExecutions: 1300.0},
		OverLappedLevel{Price: 105.1, AccumulatedExecutions: 1300.125},
		OverLappedLevel{Price: 104.1, AccumulatedExecutions: 1100.125},
	}
	//simple case for exec
	maxExec := LevelIndex{}
	leastSurplus := SurplusIndex{}
	p, i := getTradePrice(&overlap, &maxExec, &leastSurplus, 0)
	assert.Equal(107.1, p)
	assert.Equal(3, i)
	overlap = []OverLappedLevel{
		OverLappedLevel{Price: 110.1, AccumulatedExecutions: 500.0, BuySellSurplus: -800.0},
		OverLappedLevel{Price: 109.1, AccumulatedExecutions: 300.0, BuySellSurplus: -700.0},
		OverLappedLevel{Price: 108.1, AccumulatedExecutions: 1300.125, BuySellSurplus: -600.0},
		OverLappedLevel{Price: 107.1, AccumulatedExecutions: 1300.125, BuySellSurplus: 500.0},
		OverLappedLevel{Price: 106.1, AccumulatedExecutions: 1300.0, BuySellSurplus: 800.0},
		OverLappedLevel{Price: 105.1, AccumulatedExecutions: 1300.125, BuySellSurplus: 1800.0},
		OverLappedLevel{Price: 104.1, AccumulatedExecutions: 1100.125, BuySellSurplus: 2800.0},
	}
	//simple case for surplus
	maxExec.clear()
	leastSurplus.clear()
	p, i = getTradePrice(&overlap, &maxExec, &leastSurplus, 0)
	assert.Equal(107.1, p)
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
	me := NewMatchEng(100, 0.5)
	book := NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 100.0, 150)
	book.InsertOrder("2", SELLSIDE, 100, 98.0, 250)
	book.InsertOrder("3", SELLSIDE, 101, 97.0, 50)
	book.InsertOrder("4", BUYSIDE, 101, 98.0, 150)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100)
	assert.Equal(98.0, p)
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
	book.InsertOrder("1", BUYSIDE, 100, 100.0, 150)
	book.InsertOrder("2", SELLSIDE, 100, 96.0, 100)
	book.InsertOrder("3", SELLSIDE, 101, 97.0, 200)
	book.InsertOrder("4", BUYSIDE, 101, 99.0, 50)
	book.InsertOrder("5", BUYSIDE, 102, 97.0, 300)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100)
	assert.Equal(97.0, p)
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
	book.InsertOrder("1", BUYSIDE, 100, 102.0, 300)
	book.InsertOrder("2", BUYSIDE, 101, 100.0, 100)
	book.InsertOrder("3", SELLSIDE, 101, 98.0, 250)
	book.InsertOrder("4", BUYSIDE, 101, 99.0, 200)
	book.InsertOrder("5", BUYSIDE, 102, 98.0, 300)
	book.InsertOrder("6", SELLSIDE, 102, 97.0, 250)
	book.InsertOrder("7", SELLSIDE, 103, 96.0, 1000)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100)
	assert.Equal(96.0, p)
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
	book.InsertOrder("1", BUYSIDE, 100, 102.0, 30)
	book.InsertOrder("2", BUYSIDE, 101, 101.0, 10)
	book.InsertOrder("3", SELLSIDE, 101, 98.0, 10)
	book.InsertOrder("4", BUYSIDE, 101, 99.0, 50)
	book.InsertOrder("5", BUYSIDE, 102, 96.0, 15)
	book.InsertOrder("6", SELLSIDE, 102, 97.0, 50)
	book.InsertOrder("7", SELLSIDE, 103, 95.0, 50)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100)
	assert.Equal(97.0, p)
	assert.Equal(4, i)

	/* 	5. choose the lowest for all the same value of sell surplus imbalance (Step 3)
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
	book.InsertOrder("1", BUYSIDE, 100, 102.0, 10)
	book.InsertOrder("2", BUYSIDE, 101, 97.0, 10)
	book.InsertOrder("3", SELLSIDE, 101, 95.0, 50)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100)
	assert.Equal(95.0, p)
	assert.Equal(2, i)
	/*		--------------------------------------------------------------
	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	20             102      50     50     20           30*
	20             101             50     20           30
	20             100             50     20           30
	20             99              50     20           30
	20             98              50     20           30
	20     10      97              50     20           30
	10             96              50     10           40
	10     10      95              50     10           40
	*/
	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", SELLSIDE, 100, 97.0, 10)
	book.InsertOrder("2", SELLSIDE, 101, 95.0, 10)
	book.InsertOrder("3", BUYSIDE, 101, 102.0, 50)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 100)
	assert.Equal(102.0, p)
	assert.Equal(0, i)

	/* 	6. choose the closest to the last trade price 99 (Step 4)
	   	--------------------------------------------------------------
	   	SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
	   	50             100      25     25     25           -25*
	   	50             99              25     25           -25
	   	50     25      98              25     25           -25
	   	25             97       25     50     25           25
	   	25             96              50     25           25
	   	25     25      95              50     25           25
	*/

	book = NewOrderBookOnULList(4096, 16)
	book.InsertOrder("1", BUYSIDE, 100, 100.0, 25)
	book.InsertOrder("4", SELLSIDE, 101, 98.0, 25)
	book.InsertOrder("2", BUYSIDE, 101, 97.0, 25)
	book.InsertOrder("3", SELLSIDE, 101, 95.0, 25)
	book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	prepareMatch(&me.overLappedLevel)
	p, i = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 99)
	assert.Equal(100.0, p)
	assert.Equal(0, i)
}
func Test_calLeastSurplus(t *testing.T) {
	assert := assert.New(t)
	overlap := []OverLappedLevel{
		OverLappedLevel{AccumulatedExecutions: 500.0, BuySellSurplus: -800.0},
		OverLappedLevel{AccumulatedExecutions: 300.0, BuySellSurplus: -700.0},
		OverLappedLevel{AccumulatedExecutions: 1300.125, BuySellSurplus: -600.0},
		OverLappedLevel{AccumulatedExecutions: 1300.125, BuySellSurplus: -500.0},
		OverLappedLevel{AccumulatedExecutions: 1300.0, BuySellSurplus: 300.0},
		OverLappedLevel{AccumulatedExecutions: 1300.125, BuySellSurplus: 400.0},
		OverLappedLevel{AccumulatedExecutions: 1300.125, BuySellSurplus: -500.0},
		OverLappedLevel{AccumulatedExecutions: 1200.125, BuySellSurplus: 500.0},
		OverLappedLevel{AccumulatedExecutions: 1300.125, BuySellSurplus: 500.0},
	}
	me := NewMatchEng(100, 0.5)
	maxExec := me.maxExec
	leastSurplus := me.leastSurplus
	calMaxExec(&overlap, &maxExec)
	calLeastSurplus(&overlap, &maxExec, &leastSurplus)
	assert.Equal([]int{5}, leastSurplus.index)
	assert.Equal(400.0, leastSurplus.value)
	assert.Equal([]float64{400.0}, leastSurplus.surplus)

	overlap2 := overlap[:4]
	maxExec.clear()
	leastSurplus.clear()
	calMaxExec(&overlap2, &maxExec)
	calLeastSurplus(&overlap2, &maxExec, &leastSurplus)
	assert.Equal([]int{3}, leastSurplus.index)
	assert.Equal(500.0, leastSurplus.value)
	assert.Equal([]float64{-500.0}, leastSurplus.surplus)

	overlap2 = overlap[6:]
	maxExec.clear()
	leastSurplus.clear()
	calMaxExec(&overlap2, &maxExec)
	calLeastSurplus(&overlap2, &maxExec, &leastSurplus)
	assert.Equal([]int{0, 2}, leastSurplus.index)
	assert.Equal(500.0, leastSurplus.value)
	assert.Equal([]float64{-500.0, 500}, leastSurplus.surplus)
}

func TestMatchEng_fillOrders(t *testing.T) {
	assert := assert.New(t)
	me := NewMatchEng(100, 0.5)
	me.lastTradePrice = 99.99
	me.overLappedLevel = []OverLappedLevel{OverLappedLevel{Price: 100,
		BuyOrders: []OrderPart{
			OrderPart{"2", 100, 80},
			OrderPart{"1", 100, 70},
			OrderPart{"4", 100, 50},
			OrderPart{"3", 100, 100},
		},
		SellOrders: []OrderPart{
			OrderPart{"9", 100, 60},
			OrderPart{"8", 100, 70},
			OrderPart{"7", 100, 50},
			OrderPart{"6", 100, 100},
		},
	}}
	prepareMatch(&me.overLappedLevel)
	t.Log(me.overLappedLevel)
	assert.Equal(280.0, me.overLappedLevel[0].AccumulatedExecutions)
	me.fillOrders(0, 0)
	assert.Equal(20.0, me.overLappedLevel[0].BuyTotal)
	assert.Equal(0.0, me.overLappedLevel[0].SellTotal)
	t.Log(me.trades)
	assert.Equal([]Trade{
		Trade{"6", 99.99, 70.0, "1"},
		Trade{"6", 99.99, 30.0, "2"},
		Trade{"7", 99.99, 50.0, "2"},
		Trade{"8", 99.99, 70.0, "3"},
		Trade{"9", 99.99, 30.0, "3"},
		Trade{"9", 99.99, 30.0, "4"},
	}, me.trades)
}

func TestMatchEng_reserveQty(t *testing.T) {
	me := NewMatchEng(100, 0.5)
	assert := assert.New(t)
	ords := []OrderPart{
		OrderPart{"1", 100, 90},
	}
	assert.True(me.reserveQty(70, ords))
	assert.Equal(70.0, ords[0].qty)
	ords = []OrderPart{
		OrderPart{"1", 100, 90},
		OrderPart{"2", 100, 90},
		OrderPart{"3", 100, 90},
	}

	assert.True(me.reserveQty(90, ords))

}

func Test_allocateResidual(t *testing.T) {
	assert := assert.New(t)
	orders := []OrderPart{
		OrderPart{"1", 100, 90},
	}
	toAlloc := 50.0
	assert.True(allocateResidual(&toAlloc, orders, 0.5))
	assert.Equal(50.0, orders[0].qty)
	assert.Equal(0.0, toAlloc)
}
