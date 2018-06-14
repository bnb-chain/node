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
