package matcheng

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/node/common/upgrade"
)

func Test_dropRedundantQty(t *testing.T) {
	assert := assert.New(t)

	assert.Error(dropRedundantQty([]OrderPart{}, -1, 5))
	assert.Error(dropRedundantQty([]OrderPart{}, 100, 5))

	orders := []OrderPart{
		{"1", 100, 1000, 100, 900},
	}
	err := dropRedundantQty(orders, 1000, 5)
	assert.Error(err)
	assert.Equal("no enough quantity can be dropped, toDropQty=1000, totalQty=900", err.Error())

	assert.NoError(dropRedundantQty(orders, 400, 5))
	assert.Equal(int64(500), orders[0].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 100, 900},
	}
	assert.NoError(dropRedundantQty(orders, 900, 5))
	assert.Equal(int64(0), orders[0].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 100, 1000, 700, 300},
	}
	assert.NoError(dropRedundantQty(orders, 400, 5))
	assert.Equal(int64(100), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(100), orders[1].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 100, 1000, 700, 300},
	}
	assert.NoError(dropRedundantQty(orders, 600, 5))
	assert.Equal(int64(0), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(0), orders[1].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 100, 1000, 600, 400},
	}
	assert.NoError(dropRedundantQty(orders, 600, 5))
	assert.Equal(int64(45), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(55), orders[1].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 101, 1000, 700, 300},
	}
	assert.NoError(dropRedundantQty(orders, 200, 5))
	assert.Equal(int64(300), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(100), orders[1].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 101, 1000, 700, 300},
	}
	assert.NoError(dropRedundantQty(orders, 400, 5))
	assert.Equal(int64(200), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(0), orders[1].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 101, 1000, 700, 300},
	}
	assert.NoError(dropRedundantQty(orders, 600, 5))
	assert.Equal(int64(0), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(0), orders[1].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 101, 1000, 700, 300},
		{"3", 101, 1000, 700, 300},
	}
	assert.NoError(dropRedundantQty(orders, 700, 5))
	assert.Equal(int64(200), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(0), orders[1].nxtTrade)
	assert.Equal("2", orders[1].Id)
	assert.Equal(int64(0), orders[2].nxtTrade)

	orders = []OrderPart{
		{"1", 100, 1000, 800, 200},
		{"2", 100, 1000, 700, 300},
		{"3", 101, 1000, 600, 400},
		{"4", 101, 1000, 500, 500},
		{"5", 102, 1000, 400, 600},
	}
	assert.NoError(dropRedundantQty(orders, 700, 5))
	assert.Equal(int64(200), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(300), orders[1].nxtTrade)
	assert.Equal("2", orders[1].Id)
	assert.Equal(int64(360), orders[2].nxtTrade)
	assert.Equal("3", orders[2].Id)
	assert.Equal(int64(440), orders[3].nxtTrade)
	assert.Equal("4", orders[3].Id)

	orders = []OrderPart{
		{"1", 100, 100, 75, 25},
		{"2", 100, 100, 65, 35},
		{"3", 101, 100, 55, 45},
		{"4", 101, 100, 45, 55},
		{"5", 102, 100, 35, 65},
	}
	assert.NoError(dropRedundantQty(orders, 70, 10))
	assert.Equal(int64(25), orders[0].nxtTrade)
	assert.Equal("1", orders[0].Id)
	assert.Equal(int64(35), orders[1].nxtTrade)
	assert.Equal("2", orders[1].Id)
	assert.Equal(int64(45), orders[2].nxtTrade)
	assert.Equal("3", orders[2].Id)
	assert.Equal(int64(50), orders[3].nxtTrade)
	assert.Equal("4", orders[3].Id)
}

func TestMatchEng_DropRedundantOrders(t *testing.T) {
	assert := assert.New(t)
	me := NewMatchEng(DefaultPairSymbol, 100, 5, 0.05)
	me.overLappedLevel = []OverLappedLevel{{
		Price: 1000,
		BuyOrders: []OrderPart{
			{"1", 100, 100, 0, 0},
		},
		SellOrders: []OrderPart{
			{"2", 100, 100, 0, 0},
		},
	},
	}
	prepareMatch(&me.overLappedLevel)
	_, index := getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, me.LastTradePrice, me.PriceLimitPct)
	t.Log(me.overLappedLevel)
	assert.NoError(me.dropRedundantQty(index))
	assert.Equal([]OverLappedLevel{
		{
			Price: 1000,
			BuyOrders: []OrderPart{
				{"1", 100, 100, 0, 100},
			},
			SellOrders: []OrderPart{
				{"2", 100, 100, 0, 100},
			},
			SellTotal:             100,
			AccumulatedSell:       100,
			BuyTotal:              100,
			AccumulatedBuy:        100,
			AccumulatedExecutions: 100,
			BuySellSurplus:        0,
		},
	}, me.overLappedLevel)

	//
	me.overLappedLevel = []OverLappedLevel{{
		Price: 1000,
		BuyOrders: []OrderPart{
			{"1", 100, 300, 0, 0},
			{"3", 100, 400, 0, 0},
		},
		SellOrders: []OrderPart{
			{"2", 100, 300, 0, 0},
			{"4", 100, 200, 0, 0},
		},
	},
	}
	prepareMatch(&me.overLappedLevel)
	_, index = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, me.LastTradePrice, me.PriceLimitPct)
	t.Log(me.overLappedLevel)
	assert.NoError(me.dropRedundantQty(index))
	assert.Equal([]OverLappedLevel{
		{
			Price: 1000,
			BuyOrders: []OrderPart{
				{"1", 100, 300, 0, 215},
				{"3", 100, 400, 0, 285},
			},
			SellOrders: []OrderPart{
				{"2", 100, 300, 0, 300},
				{"4", 100, 200, 0, 200},
			},
			SellTotal:             500,
			AccumulatedSell:       500,
			BuyTotal:              700,
			AccumulatedBuy:        700,
			AccumulatedExecutions: 500,
			BuySellSurplus:        200,
		},
	}, me.overLappedLevel)

	//
	me.overLappedLevel = []OverLappedLevel{{
		Price: 1000,
		BuyOrders: []OrderPart{
			{"1", 100, 100, 0, 0},
			{"3", 100, 200, 0, 0},
		},
		SellOrders: []OrderPart{
			{"2", 100, 300, 0, 0},
			{"4", 100, 400, 0, 0},
			{"6", 101, 400, 0, 0},
		},
	}}
	prepareMatch(&me.overLappedLevel)
	_, index = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, me.LastTradePrice, me.PriceLimitPct)
	t.Log(me.overLappedLevel)
	assert.NoError(me.dropRedundantQty(index))
	assert.Equal([]OverLappedLevel{
		{
			Price: 1000,
			BuyOrders: []OrderPart{
				{"1", 100, 100, 0, 100},
				{"3", 100, 200, 0, 200},
			},
			SellOrders: []OrderPart{
				{"2", 100, 300, 0, 130},
				{"4", 100, 400, 0, 170},
				{"6", 101, 400, 0, 0},
			},
			SellTotal:             1100,
			AccumulatedSell:       1100,
			BuyTotal:              300,
			AccumulatedBuy:        300,
			AccumulatedExecutions: 300,
			BuySellSurplus:        -800,
		},
	}, me.overLappedLevel)

	//
	me.overLappedLevel = []OverLappedLevel{{
		Price: 1200,
		BuyOrders: []OrderPart{
			{"1", 100, 300, 0, 0},
		}}, {
		Price: 1100,
		BuyOrders: []OrderPart{
			{"3", 100, 200, 0, 0},
		}}, {
		Price: 1000,
		BuyOrders: []OrderPart{
			{"5", 101, 100, 0, 0},
		}}, {
		Price: 900,
		SellOrders: []OrderPart{
			{"2", 100, 1000, 0, 0},
		}},
	}
	prepareMatch(&me.overLappedLevel)
	_, index = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 1000, me.PriceLimitPct)
	t.Log(me.overLappedLevel)
	assert.NoError(me.dropRedundantQty(index))
	assert.Equal([]OverLappedLevel{{
		Price: 1200,
		BuyOrders: []OrderPart{
			{"1", 100, 300, 0, 300},
		},
		SellTotal:             0,
		AccumulatedSell:       1000,
		BuyTotal:              300,
		AccumulatedBuy:        300,
		AccumulatedExecutions: 300,
		BuySellSurplus:        -700,
	}, {
		Price: 1100,
		BuyOrders: []OrderPart{
			{"3", 100, 200, 0, 200},
		},
		SellTotal:             0,
		AccumulatedSell:       1000,
		BuyTotal:              200,
		AccumulatedBuy:        500,
		AccumulatedExecutions: 500,
		BuySellSurplus:        -500,
	}, {
		Price: 1000,
		BuyOrders: []OrderPart{
			{"5", 101, 100, 0, 100},
		},
		SellTotal:             0,
		AccumulatedSell:       1000,
		BuyTotal:              100,
		AccumulatedBuy:        600,
		AccumulatedExecutions: 600,
		BuySellSurplus:        -400,
	}, {
		Price: 900,
		SellOrders: []OrderPart{
			{"2", 100, 1000, 0, 600},
		},
		SellTotal:             1000,
		AccumulatedSell:       1000,
		BuyTotal:              0,
		AccumulatedBuy:        600,
		AccumulatedExecutions: 600,
		BuySellSurplus:        -400,
	}}, me.overLappedLevel)

	//
	me.overLappedLevel = []OverLappedLevel{{
		Price: 1200,
		BuyOrders: []OrderPart{
			{"1", 100, 1000, 0, 0},
		}}, {
		Price: 1100,
		SellOrders: []OrderPart{
			{"2", 100, 100, 0, 0},
		}}, {
		Price: 1000,
		SellOrders: []OrderPart{
			{"4", 101, 200, 0, 0},
		}}, {
		Price: 900,
		SellOrders: []OrderPart{
			{"6", 101, 300, 0, 0},
		}},
	}
	prepareMatch(&me.overLappedLevel)
	_, index = getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, 1000, me.PriceLimitPct)
	t.Log(me.overLappedLevel)
	assert.NoError(me.dropRedundantQty(index))
	assert.Equal([]OverLappedLevel{{
		Price: 1200,
		BuyOrders: []OrderPart{
			{"1", 100, 1000, 0, 600},
		},
		SellTotal:             0,
		AccumulatedSell:       600,
		BuyTotal:              1000,
		AccumulatedBuy:        1000,
		AccumulatedExecutions: 600,
		BuySellSurplus:        400,
	}, {
		Price: 1100,
		SellOrders: []OrderPart{
			{"2", 100, 100, 0, 100},
		},
		SellTotal:             100,
		AccumulatedSell:       600,
		BuyTotal:              0,
		AccumulatedBuy:        1000,
		AccumulatedExecutions: 600,
		BuySellSurplus:        400,
	}, {
		Price: 1000,
		SellOrders: []OrderPart{
			{"4", 101, 200, 0, 200},
		},
		SellTotal:             200,
		AccumulatedSell:       500,
		BuyTotal:              0,
		AccumulatedBuy:        1000,
		AccumulatedExecutions: 500,
		BuySellSurplus:        500,
	}, {
		Price: 900,
		SellOrders: []OrderPart{
			{"6", 101, 300, 0, 300},
		},
		SellTotal:             300,
		AccumulatedSell:       300,
		BuyTotal:              0,
		AccumulatedBuy:        1000,
		AccumulatedExecutions: 300,
		BuySellSurplus:        700,
	}}, me.overLappedLevel)
}

func Test_calcFillQty(t *testing.T) {
	assert := assert.New(t)
	takers := []*OrderPart{
		{"1", 100, 1800, 900, 900},
	}
	toFillQty := make([]int64, len(takers))
	calcFillQty(toFillQty, 600, takers, []int64{900}, 900, 5)
	assert.Equal(int64(900), takers[0].nxtTrade)
	assert.Equal([]int64{600}, toFillQty)

	// check takers not modified
	takers = []*OrderPart{
		{"1", 100, 900, 0, 900},
		{"2", 100, 300, 0, 300},
		{"3", 100, 600, 0, 600},
	}
	toFillQty = make([]int64, len(takers))
	calcFillQty(toFillQty, 600, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal("1", takers[0].Id)
	assert.Equal(int64(900), takers[0].nxtTrade)
	assert.Equal("2", takers[1].Id)
	assert.Equal(int64(300), takers[1].nxtTrade)
	assert.Equal(int64(600), takers[2].nxtTrade)
	assert.Equal([]int64{300, 100, 200}, toFillQty)

	calcFillQty(toFillQty, 500, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal([]int64{255, 80, 165}, toFillQty)

	calcFillQty(toFillQty, 25, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal([]int64{15, 5, 5}, toFillQty)

	calcFillQty(toFillQty, 35, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal([]int64{20, 5, 10}, toFillQty)

	takers = []*OrderPart{
		{"1", 100, 900, 0, 900},
		{"2", 100, 900, 0, 900},
		{"3", 100, 900, 0, 900},
	}
	calcFillQty(toFillQty, 700, takers, []int64{900, 900, 900}, 2700, 5)
	assert.Equal([]int64{235, 235, 230}, toFillQty)

	takers = []*OrderPart{
		{"1", 100, 1, 0, 1},
		{"2", 100, 10, 0, 10},
		{"3", 100, 6, 0, 6},
	}
	calcFillQty(toFillQty, 15, takers, []int64{1, 10, 6}, 17, 5)
	assert.Equal([]int64{1, 9, 5}, toFillQty)

	takers = []*OrderPart{
		{"1", 100, 10, 0, 10},
		{"2", 100, 5, 0, 5},
		{"3", 100, 50, 0, 50},
	}
	calcFillQty(toFillQty, 35, takers, []int64{10, 5, 50}, 65, 5)
	assert.Equal([]int64{10, 0, 25}, toFillQty)
}

func TestMatchEng_determineTakerSide(t *testing.T) {
	assert := assert.New(t)
	me := NewMatchEng(DefaultPairSymbol, 100, 5, 0.05)
	me.overLappedLevel = []OverLappedLevel{{
		Price: 1200,
		BuyOrders: []OrderPart{
			{"1", 100, 100, 0, 100},
			{"3", 100, 100, 0, 100},
		},
		SellOrders: []OrderPart{
			{"2", 99, 100, 0, 100},
			{"4", 99, 100, 0, 100},
			{"6", 100, 100, 0, 100},
		},
	}, {
		Price: 1100,
		// BuyOrders is empty
		BuyOrders: []OrderPart{},
		SellOrders: []OrderPart{
			{"8", 99, 100, 0, 100},
		},
	}, {
		Price: 1000,
		BuyOrders: []OrderPart{
			{"5", 99, 100, 0, 100},
			{"7", 99, 100, 0, 100},
			{"9", 100, 100, 0, 100},
		},
		// SellOrders is nil
	}, {
		Price: 900,
		BuyOrders: []OrderPart{
			{"11", 99, 100, 0, 100},
		},
		SellOrders: []OrderPart{
			{"10", 100, 100, 0, 100},
		},
	}}

	checkAndClear := func(l *OverLappedLevel, buyTakerStartIdx int, buyMakerTotal int64, sellTakerStartIdx int, sellMakerTotal int64) {
		assert.Equal(buyTakerStartIdx, l.BuyTakerStartIdx)
		assert.Equal(buyMakerTotal, l.BuyMakerTotal)
		assert.Equal(sellTakerStartIdx, l.SellTakerStartIdx)
		assert.Equal(sellMakerTotal, l.SellMakerTotal)
		l.BuyTakerStartIdx, l.BuyMakerTotal, l.SellTakerStartIdx, l.SellMakerTotal = 0, 0, 0, 0
	}

	takerSide, err := me.determineTakerSide(99, 0)
	assert.NoError(err)
	assert.Equal(BUYSIDE, takerSide)
	checkAndClear(&me.overLappedLevel[0], 0, 0, 2, 200)
	checkAndClear(&me.overLappedLevel[1], 0, 0, 1, 100)
	checkAndClear(&me.overLappedLevel[2], 0, 0, 0, 0)
	checkAndClear(&me.overLappedLevel[3], 0, 0, 0, 0)

	takerSide, err = me.determineTakerSide(99, 1)
	assert.NoError(err)
	assert.Equal(BUYSIDE, takerSide)
	checkAndClear(&me.overLappedLevel[0], 0, 0, 0, 0)
	checkAndClear(&me.overLappedLevel[1], 0, 0, 1, 100)
	checkAndClear(&me.overLappedLevel[2], 0, 0, 0, 0)
	checkAndClear(&me.overLappedLevel[3], 0, 0, 0, 0)

	takerSide, err = me.determineTakerSide(99, 2)
	assert.NoError(err)
	assert.Equal(SELLSIDE, takerSide)
	checkAndClear(&me.overLappedLevel[0], 0, 0, 0, 0)
	checkAndClear(&me.overLappedLevel[1], 0, 0, 0, 0)
	checkAndClear(&me.overLappedLevel[2], 2, 200, 0, 0)
	checkAndClear(&me.overLappedLevel[3], 0, 0, 0, 0)

	takerSide, err = me.determineTakerSide(99, 3)
	assert.NoError(err)
	assert.Equal(SELLSIDE, takerSide)
	checkAndClear(&me.overLappedLevel[0], 0, 0, 0, 0)
	checkAndClear(&me.overLappedLevel[1], 0, 0, 0, 0)
	checkAndClear(&me.overLappedLevel[2], 2, 200, 0, 0)
	checkAndClear(&me.overLappedLevel[3], 1, 100, 0, 0)

	me.overLappedLevel = []OverLappedLevel{{
		Price: 1200,
		BuyOrders: []OrderPart{
			{"1", 100, 100, 0, 100},
		},
		SellOrders: []OrderPart{
			{"2", 100, 100, 0, 100},
		},
	}}
	takerSide, err = me.determineTakerSide(99, 0)
	assert.NoError(err)
	assert.Equal(BUYSIDE, takerSide)
	checkAndClear(&me.overLappedLevel[0], 0, 0, 0, 0)

	me.overLappedLevel = []OverLappedLevel{{
		Price: 1200,
		BuyOrders: []OrderPart{
			{"1", 99, 100, 0, 100},
		},
		SellOrders: []OrderPart{
			{"2", 99, 100, 0, 100},
		},
	}}
	takerSide, err = me.determineTakerSide(99, 0)
	assert.EqualError(err, "both buy side and sell side have maker orders.")
	assert.Equal(UNKNOWN, takerSide)
}

func Test_mergeOneTakerLevel(t *testing.T) {
	assert := assert.New(t)

	//
	merged := NewMergedPriceLevel(100)
	overlapped := OverLappedLevel{
		Price:             110,
		BuyOrders:         []OrderPart{},
		SellOrders:        []OrderPart{},
		BuyTakerStartIdx:  0,
		SellTakerStartIdx: 0,
	}
	mergeOneTakerLevel(BUYSIDE, &overlapped, merged)
	assert.Equal(0, len(merged.orders))
	assert.Equal(int64(0), merged.totalQty)
	mergeOneTakerLevel(SELLSIDE, &overlapped, merged)
	assert.Equal(0, len(merged.orders))
	assert.Equal(int64(0), merged.totalQty)

	//
	overlapped = OverLappedLevel{
		Price: 110,
		BuyOrders: []OrderPart{
			{"1", 100, 1000, 0, 0},
		},
		BuyTakerStartIdx: 0,
	}
	mergeOneTakerLevel(BUYSIDE, &overlapped, merged)
	assert.Equal(0, len(merged.orders))
	assert.Equal(int64(0), merged.totalQty)

	//
	overlapped = OverLappedLevel{
		Price: 110,
		BuyOrders: []OrderPart{
			{"1", 99, 200, 0, 200},
			{"2", 100, 500, 0, 500},
			{"3", 100, 1000, 0, 1000},
		},
		BuyTakerStartIdx: 1,
	}
	mergeOneTakerLevel(BUYSIDE, &overlapped, merged)
	assert.EqualValues([]*OrderPart{
		{"3", 100, 1000, 0, 1000},
		{"2", 100, 500, 0, 500},
	}, merged.orders)
	assert.Equal(int64(1500), merged.totalQty)

	//
	merged = NewMergedPriceLevel(100)
	overlapped = OverLappedLevel{
		Price: 110,
		BuyOrders: []OrderPart{
			{"1", 99, 1000, 0, 200},
			{"2", 99, 1000, 0, 0},
			{"3", 100, 300, 0, 300},
			{"4", 100, 100, 0, 0},
			{"5", 100, 400, 0, 300},
		},
		BuyTakerStartIdx: 2,
	}
	mergeOneTakerLevel(BUYSIDE, &overlapped, merged)
	assert.Equal([]*OrderPart{
		{"5", 100, 400, 0, 300},
		{"3", 100, 300, 0, 300},
	}, merged.orders)
	assert.Equal(int64(600), merged.totalQty)
}

func Test_mergeTakerSideOrders(t *testing.T) {
	type args struct {
		side           int8
		concludedPrice int64
		overlapped     []OverLappedLevel
		tradePriceIdx  int
	}
	tests := []struct {
		name string
		args args
		want TakerSideOrders
	}{{
		name: "buySideIsTakerSide",
		args: args{
			side:           BUYSIDE,
			concludedPrice: 100,
			overlapped: []OverLappedLevel{{
				Price: 110,
				BuyOrders: []OrderPart{
					{"1", 99, 200, 100, 100},
					{"3", 100, 100, 0, 100},
					{"5", 100, 500, 0, 500},
				},
				BuyTakerStartIdx: 1,
				SellOrders: []OrderPart{
					{"2", 100, 1000, 0, 1000},
				},
				SellTakerStartIdx: 0,
			}, {
				Price: 105,
				BuyOrders: []OrderPart{
					{"7", 99, 200, 100, 100},
				},
				BuyTakerStartIdx:  1,
				SellTakerStartIdx: 0,
			}, {
				Price: 100,
				BuyOrders: []OrderPart{
					{"9", 100, 200, 0, 200},
					{"11", 100, 100, 0, 100},
				},
				BuyTakerStartIdx:  0,
				SellTakerStartIdx: 0,
			}},
			tradePriceIdx: 2,
		},
		want: TakerSideOrders{
			&MergedPriceLevel{
				price: 100,
				orders: []*OrderPart{
					{"5", 100, 500, 0, 500},
					{"3", 100, 100, 0, 100},
					{"9", 100, 200, 0, 200},
					{"11", 100, 100, 0, 100},
				},
				totalQty: 900,
			},
		},
	}, {
		name: "sellSideIsTakerSide",
		args: args{
			side:           SELLSIDE,
			concludedPrice: 110,
			overlapped: []OverLappedLevel{{
				Price: 110,
				BuyOrders: []OrderPart{
					{"1", 99, 200, 100, 100},
				},
				BuyTakerStartIdx: 1,
				SellOrders: []OrderPart{
					{"2", 99, 1000, 0, 1000},
				},
				SellTakerStartIdx: 1,
			}, {
				Price:            105,
				BuyTakerStartIdx: 0,
				SellOrders: []OrderPart{
					{"4", 99, 200, 0, 200},
					{"6", 100, 1000, 0, 1000},
				},
				SellTakerStartIdx: 1,
			}, {
				Price: 100,
				SellOrders: []OrderPart{
					{"8", 100, 200, 0, 200},
					{"10", 100, 300, 0, 300},
				},
				BuyTakerStartIdx:  0,
				SellTakerStartIdx: 0,
			}},
			tradePriceIdx: 0,
		},
		want: TakerSideOrders{
			&MergedPriceLevel{
				price: 110,
				orders: []*OrderPart{
					{"10", 100, 300, 0, 300},
					{"8", 100, 200, 0, 200},
					{"6", 100, 1000, 0, 1000},
				},
				totalQty: 1500,
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeTakerSideOrders(tt.args.side, tt.args.concludedPrice, tt.args.overlapped, tt.args.tradePriceIdx)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestMatchEng_fillOrdersNew(t *testing.T) {
	assert := assert.New(t)
	// 1. buy side is maker side
	me := NewMatchEng("AAA_BNB", 100, 5, 0.05)
	makerSideOrders := []OrderPart{
		{"1", 99, 1000, 700, 300},
		{"3", 99, 1000, 900, 100},
		{"5", 100, 1000, 700, 300},
		{"7", 99, 1000, 800, 200},
		{"9", 100, 1000, 200, 100},
	}
	me.overLappedLevel = []OverLappedLevel{{
		Price:            110,
		BuyOrders:        makerSideOrders[:3],
		BuyTakerStartIdx: 2,
		BuyMakerTotal:    400,
	}, {
		Price:            100,
		BuyOrders:        makerSideOrders[3:],
		BuyTakerStartIdx: 1,
		BuyMakerTotal:    200,
	}}
	takerSideOrders := TakerSideOrders{
		&MergedPriceLevel{
			price: 100,
			orders: []*OrderPart{
				{"8", 100, 1000, 800, 200},
				{"6", 100, 800, 500, 300},
				{"2", 100, 600, 200, 400},
				{"4", 100, 400, 300, 100},
			},
			totalQty: 1000,
		},
	}

	me.fillOrdersNew(SELLSIDE, takerSideOrders, 1, 100, 10)
	assert.Equal([]Trade{
		{"8", 110, 80, 780, 880, "1", SellTaker, nil, nil},
		{"6", 110, 120, 900, 620, "1", SellTaker, nil, nil},
		{"2", 110, 100, 1000, 300, "1", SellTaker, nil, nil},
		{"2", 110, 60, 960, 360, "3", SellTaker, nil, nil},
		{"4", 110, 40, 1000, 340, "3", SellTaker, nil, nil},

		{"8", 100, 120, 820, 1000, "5", BuySurplus, nil, nil},
		{"6", 100, 180, 1000, 800, "5", BuySurplus, nil, nil},
		{"2", 100, 200, 1000, 560, "7", SellTaker, nil, nil},
		{"2", 100, 40, 240, 600, "9", BuySurplus, nil, nil},
		{"4", 100, 60, 300, 400, "9", BuySurplus, nil, nil},
	}, me.Trades)
	assert.Equal([]OrderPart{
		{"1", 99, 1000, 1000, 0},
		{"3", 99, 1000, 1000, 0},
		{"5", 100, 1000, 1000, 0},
		{"7", 99, 1000, 1000, 0},
		{"9", 100, 1000, 300, 0},
	}, makerSideOrders)
	assert.Equal([]*OrderPart{
		{"8", 100, 1000, 1000, 0},
		{"6", 100, 800, 800, 0},
		{"2", 100, 600, 600, 0},
		{"4", 100, 400, 400, 0},
	}, takerSideOrders.orders)

	// 2. sell side is maker side
	me = NewMatchEng("AAA_BNB", 100, 5, 0.05)
	makerSideOrders = []OrderPart{
		{"2", 99, 1000, 700, 300},
		{"4", 99, 1000, 800, 200},
		{"6", 99, 1000, 900, 100},
		{"8", 99, 1000, 900, 100},
		{"10", 99, 1000, 900, 100},
		{"12", 100, 1000, 800, 200},
	}
	me.overLappedLevel = []OverLappedLevel{{
		Price:             100,
		SellOrders:        makerSideOrders[3:],
		SellTakerStartIdx: 2,
		SellMakerTotal:    200,
	}, {
		Price:             95,
		SellOrders:        makerSideOrders[1:3],
		SellTakerStartIdx: 2,
		SellMakerTotal:    300,
	}, {
		Price:             90,
		SellOrders:        makerSideOrders[:1],
		SellTakerStartIdx: 1,
		SellMakerTotal:    300,
	}}
	takerSideOrders = TakerSideOrders{
		&MergedPriceLevel{
			price: 100,
			orders: []*OrderPart{
				{"1", 100, 600, 0, 600},
				{"3", 100, 300, 0, 300},
				{"5", 100, 100, 0, 100},
			},
			totalQty: 1000,
		},
	}
	me.fillOrdersNew(BUYSIDE, takerSideOrders, 0, 100, -100)
	assert.Equal([]Trade{
		{"2", 90, 180, 180, 880, "1", BuyTaker, nil, nil},
		{"2", 90, 90, 90, 970, "3", BuyTaker, nil, nil},
		{"2", 90, 30, 30, 1000, "5", BuyTaker, nil, nil},

		{"4", 95, 180, 360, 980, "1", BuyTaker, nil, nil},
		{"4", 95, 20, 110, 1000, "3", BuyTaker, nil, nil},
		{"6", 95, 70, 180, 970, "3", BuyTaker, nil, nil},
		{"6", 95, 30, 60, 1000, "5", BuyTaker, nil, nil},

		{"8", 100, 100, 460, 1000, "1", BuyTaker, nil, nil},
		{"10", 100, 100, 560, 1000, "1", BuyTaker, nil, nil},
		{"12", 100, 40, 600, 840, "1", SellSurplus, nil, nil},
		{"12", 100, 120, 300, 960, "3", SellSurplus, nil, nil},
		{"12", 100, 40, 100, 1000, "5", SellSurplus, nil, nil},
	}, me.Trades)
	assert.Equal([]OrderPart{
		{"2", 99, 1000, 1000, 0},
		{"4", 99, 1000, 1000, 0},
		{"6", 99, 1000, 1000, 0},
		{"8", 99, 1000, 1000, 0},
		{"10", 99, 1000, 1000, 0},
		{"12", 100, 1000, 1000, 0},
	}, makerSideOrders)
	assert.Equal([]*OrderPart{
		{"1", 100, 600, 600, 0},
		{"3", 100, 300, 300, 0},
		{"5", 100, 100, 100, 0},
	}, takerSideOrders.orders)

	// 3. no maker orders
	me = NewMatchEng("AAA_BNB", 100, 5, 0.05)
	makerSideOrders = []OrderPart{
		{"2", 100, 1000, 700, 300},
		{"4", 100, 1000, 900, 100},
	}
	me.overLappedLevel = []OverLappedLevel{{
		Price:             100,
		SellOrders:        makerSideOrders[:1],
		SellTakerStartIdx: 1,
		SellMakerTotal:    0,
	}, {
		Price:             90,
		SellOrders:        makerSideOrders[1:],
		SellTakerStartIdx: 1,
		SellMakerTotal:    0,
	}}
	takerSideOrders = TakerSideOrders{
		&MergedPriceLevel{
			price: 100,
			orders: []*OrderPart{
				{"1", 100, 1000, 700, 300},
				{"3", 100, 1000, 900, 100},
			},
			totalQty: 400,
		},
	}
	me.fillOrdersNew(BUYSIDE, takerSideOrders, 0, 100, 0)
	assert.Equal([]Trade{
		{"4", 100, 100, 800, 1000, "1", Neutral, nil, nil},
		{"2", 100, 200, 1000, 900, "1", Neutral, nil, nil},
		{"2", 100, 100, 1000, 1000, "3", Neutral, nil, nil},
	}, me.Trades)
	assert.Equal([]OrderPart{
		{"2", 100, 1000, 1000, 0},
		{"4", 100, 1000, 1000, 0},
	}, makerSideOrders)
	assert.Equal([]*OrderPart{
		{"1", 100, 1000, 1000, 0},
		{"3", 100, 1000, 1000, 0},
	}, takerSideOrders.orders)
}

func TestMatchEng_Match(t *testing.T) {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP19, 1)

	assert := assert.New(t)
	me := NewMatchEng(DefaultPairSymbol, 100, 5, 0.05)
	me.Book = NewOrderBookOnULList(4, 2)
	me.Book.InsertOrder("1", SELLSIDE, 90, 100, 5)
	me.Book.InsertOrder("3", SELLSIDE, 91, 100, 10)
	me.Book.InsertOrder("5", SELLSIDE, 91, 100, 5)
	me.Book.InsertOrder("7", SELLSIDE, 91, 100, 50)
	me.Book.InsertOrder("9", SELLSIDE, 91, 110, 50)
	me.Book.InsertOrder("2", BUYSIDE, 92, 90, 5)
	me.Book.InsertOrder("4", BUYSIDE, 93, 80, 30)
	me.Book.InsertOrder("11", SELLSIDE, 100, 90, 30)
	me.Book.InsertOrder("13", SELLSIDE, 100, 80, 10)
	me.Book.InsertOrder("15", SELLSIDE, 100, 80, 40)
	me.Book.InsertOrder("17", SELLSIDE, 100, 80, 20)
	me.Book.InsertOrder("12", BUYSIDE, 100, 110, 110)
	me.Book.InsertOrder("14", BUYSIDE, 100, 100, 10)
	me.Book.InsertOrder("16", BUYSIDE, 100, 100, 20)

	upgrade.Mgr.SetHeight(100)
	me.LastMatchHeight = 99
	assert.True(me.Match(100))
	assert.Equal(4, len(me.overLappedLevel))
	assert.Equal(int64(100), me.LastTradePrice)
	assert.Equal([]Trade{
		{"13", 100, 10, 10, 10, "12", SellSurplus, nil, nil},
		{"15", 100, 40, 50, 40, "12", SellSurplus, nil, nil},
		{"17", 100, 20, 70, 20, "12", SellSurplus, nil, nil},
		{"11", 100, 30, 100, 30, "12", SellSurplus, nil, nil},
		{"1", 100, 5, 105, 5, "12", BuyTaker, nil, nil},
		{"3", 100, 5, 110, 5, "12", BuyTaker, nil, nil},
		{"3", 100, 5, 5, 10, "16", BuyTaker, nil, nil},
		{"7", 100, 15, 20, 15, "16", BuyTaker, nil, nil},
		{"7", 100, 10, 10, 25, "14", BuyTaker, nil, nil},
	}, me.Trades)
	me.DropFilledOrder()
	buys, sells := me.Book.GetAllLevels()
	assert.Equal([]PriceLevel{{
		Price: 90,
		Orders: []OrderPart{
			{"2", 92, 5, 0, 5},
		},
	}, {
		Price: 80,
		Orders: []OrderPart{
			{"4", 93, 30, 0, 30},
		},
	}}, buys)
	assert.Equal([]PriceLevel{{
		Price: 100,
		Orders: []OrderPart{
			{"5", 91, 5, 0, 5},
			{"7", 91, 50, 25, 25},
		},
	}, {
		Price: 110,
		Orders: []OrderPart{
			{"9", 91, 50, 0, 50},
		},
	}}, sells)

	//
	me = NewMatchEng(DefaultPairSymbol, 110, 10, 0.05)
	me.Book = NewOrderBookOnULList(4, 2)
	me.Book.InsertOrder("1", SELLSIDE, 90, 100, 10)
	me.Book.InsertOrder("3", SELLSIDE, 90, 100, 10)
	me.Book.InsertOrder("2", BUYSIDE, 92, 70, 5)
	me.Book.InsertOrder("4", BUYSIDE, 93, 70, 30)
	me.Book.InsertOrder("5", SELLSIDE, 99, 90, 30)
	me.Book.InsertOrder("7", SELLSIDE, 99, 80, 10)
	me.Book.InsertOrder("9", SELLSIDE, 99, 80, 40)
	me.Book.InsertOrder("11", SELLSIDE, 100, 80, 20)
	me.Book.InsertOrder("13", SELLSIDE, 100, 100, 20)
	me.Book.InsertOrder("15", SELLSIDE, 100, 100, 30)
	me.Book.InsertOrder("6", BUYSIDE, 100, 110, 40)
	me.Book.InsertOrder("8", BUYSIDE, 100, 110, 100)
	me.LastMatchHeight = 99
	assert.True(me.Match(100))
	assert.Equal(4, len(me.overLappedLevel))
	assert.Equal(int64(104), me.LastTradePrice)
	assert.Equal([]Trade{
		{"7", 80, 10, 10, 10, "8", BuyTaker, nil, nil},
		{"9", 80, 30, 40, 30, "8", BuyTaker, nil, nil},
		{"9", 80, 10, 10, 40, "6", BuyTaker, nil, nil},

		{"5", 90, 30, 70, 30, "8", BuyTaker, nil, nil},

		{"1", 100, 10, 80, 10, "8", BuyTaker, nil, nil},
		{"3", 100, 10, 90, 10, "8", BuyTaker, nil, nil},

		{"11", 104, 10, 100, 10, "8", SellSurplus, nil, nil},
		{"11", 104, 10, 20, 20, "6", SellSurplus, nil, nil},
		{"13", 104, 10, 30, 10, "6", SellSurplus, nil, nil},
		{"15", 104, 10, 40, 10, "6", SellSurplus, nil, nil},
	}, me.Trades)
	me.DropFilledOrder()
	buys, sells = me.Book.GetAllLevels()
	assert.Equal([]PriceLevel{{
		Price: 70,
		Orders: []OrderPart{
			{"2", 92, 5, 0, 0},
			{"4", 93, 30, 0, 0},
		},
	}}, buys)
	assert.Equal([]PriceLevel{{
		Price: 100,
		Orders: []OrderPart{
			{"13", 100, 20, 10, 10},
			{"15", 100, 30, 10, 20},
		},
	}}, sells)
}
