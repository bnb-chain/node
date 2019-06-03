package matcheng

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	toFillQty := calcFillQty(600, takers, []int64{900}, 900, 5)
	assert.Equal(int64(900), takers[0].nxtTrade)
	assert.Equal([]int64{600}, toFillQty)

	// check takers not modified
	takers = []*OrderPart{
		{"1", 100, 900, 0, 900},
		{"2", 100, 300, 0, 300},
		{"3", 100, 600, 0, 600},
	}
	toFillQty = calcFillQty(600, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal("1", takers[0].Id)
	assert.Equal(int64(900), takers[0].nxtTrade)
	assert.Equal("2", takers[1].Id)
	assert.Equal(int64(300), takers[1].nxtTrade)
	assert.Equal(int64(600), takers[2].nxtTrade)
	assert.Equal([]int64{300, 100, 200}, toFillQty)

	toFillQty = calcFillQty(500, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal([]int64{255, 80, 165}, toFillQty)

	toFillQty = calcFillQty(25, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal([]int64{15, 5, 5}, toFillQty)

	toFillQty = calcFillQty(35, takers, []int64{900, 300, 600}, 1800, 5)
	assert.Equal([]int64{20, 5, 10}, toFillQty)

	takers = []*OrderPart{
		{"1", 100, 900, 0, 900},
		{"2", 100, 900, 0, 900},
		{"3", 100, 900, 0, 900},
	}
	toFillQty = calcFillQty(700, takers, []int64{900, 900, 900}, 2700, 5)
	assert.Equal([]int64{235, 235, 230}, toFillQty)

	takers = []*OrderPart{
		{"1", 100, 1, 0, 1},
		{"2", 100, 10, 0, 10},
		{"3", 100, 6, 0, 6},
	}
	toFillQty = calcFillQty(15, takers, []int64{1, 10, 6}, 17, 5)
	assert.Equal([]int64{1, 9, 5}, toFillQty)

	takers = []*OrderPart{
		{"1", 100, 10, 0, 10},
		{"2", 100, 5, 0, 5},
		{"3", 100, 50, 0, 50},
	}
	toFillQty = calcFillQty(35, takers, []int64{10, 5, 50}, 65, 5)
	assert.Equal([]int64{10, 0, 25}, toFillQty)
}

func Test_mergeOnePriceLevel(t *testing.T) {
	assert := assert.New(t)

	//
	makerLevels := make([]*MergedPriceLevel, 0)
	concludedPriceLevel := NewMergedPriceLevel(100)
	overlapped := OverLappedLevel{
		Price:      110,
		BuyOrders:  []OrderPart{},
		SellOrders: []OrderPart{}}
	mergeOnePriceLevel(BUYSIDE, 100, &overlapped, &makerLevels, concludedPriceLevel)
	assert.Equal(0, len(makerLevels))
	assert.Equal(0, len(concludedPriceLevel.orders))
	mergeOnePriceLevel(SELLSIDE, 100, &overlapped, &makerLevels, concludedPriceLevel)
	assert.Equal(0, len(makerLevels))
	assert.Equal(0, len(concludedPriceLevel.orders))

	//
	overlapped = OverLappedLevel{
		Price: 110,
		BuyOrders: []OrderPart{
			{"1", 100, 1000, 0, 0},
		}}
	mergeOnePriceLevel(BUYSIDE, 100, &overlapped, &makerLevels, concludedPriceLevel)
	assert.Equal(0, len(makerLevels))
	assert.Equal(0, len(concludedPriceLevel.orders))

	//
	overlapped = OverLappedLevel{
		Price: 110,
		BuyOrders: []OrderPart{
			{"1", 99, 1000, 0, 200},
			{"2", 99, 1000, 0, 500},
			{"3", 100, 1000, 0, 0},
		}}
	mergeOnePriceLevel(BUYSIDE, 100, &overlapped, &makerLevels, concludedPriceLevel)
	assert.Equal(1, len(makerLevels))
	assert.EqualValues([]*OrderPart{
		{"2", 99, 1000, 0, 500},
		{"1", 99, 1000, 0, 200},
	}, makerLevels[0].orders)
	assert.Equal(0, len(concludedPriceLevel.orders))

	//
	makerLevels = make([]*MergedPriceLevel, 0)
	concludedPriceLevel = NewMergedPriceLevel(100)
	overlapped = OverLappedLevel{
		Price: 110,
		BuyOrders: []OrderPart{
			{"1", 99, 1000, 0, 200},
			{"2", 99, 1000, 0, 0},
			{"3", 99, 1000, 0, 500},
			{"4", 100, 1000, 0, 100},
			{"5", 100, 1000, 0, 200},
		}}
	mergeOnePriceLevel(BUYSIDE, 100, &overlapped, &makerLevels, concludedPriceLevel)
	assert.Equal(1, len(makerLevels))
	assert.EqualValues([]*OrderPart{
		{"3", 99, 1000, 0, 500},
		{"1", 99, 1000, 0, 200},
	}, makerLevels[0].orders)
	assert.Equal([]*OrderPart{
		{"5", 100, 1000, 0, 200},
		{"4", 100, 1000, 0, 100},
	}, concludedPriceLevel.orders)
}

func Test_mergeSidePriceLevels(t *testing.T) {
	type args struct {
		side               int8
		height             int64
		concludedPrice     int64
		tradePriceLevelIdx int
		levels             []OverLappedLevel
	}

	tests := []struct {
		name             string
		args             args
		wantIsMakerSide  bool
		wantMergedLevels []*MergedPriceLevel
	}{{
		"buySide_oneLevel_notMaker",
		args{
			side:               BUYSIDE,
			height:             100,
			concludedPrice:     100,
			tradePriceLevelIdx: 0,
			levels: []OverLappedLevel{{
				Price: 110,
				BuyOrders: []OrderPart{
					{"1", 100, 1000, 0, 100},
				}},
			},
		},
		false,
		[]*MergedPriceLevel{{
			price: 100,
			orders: []*OrderPart{
				{"1", 100, 1000, 0, 100},
			},
			totalQty: 100,
		}},
	}, {
		"buySide_multiLevels_notMaker",
		args{
			side:               BUYSIDE,
			height:             100,
			concludedPrice:     100,
			tradePriceLevelIdx: 2,
			levels: []OverLappedLevel{{
				Price: 110,
				BuyOrders: []OrderPart{
					{"1", 100, 1000, 0, 100},
					{"2", 100, 1000, 0, 200},
				}}, {
				Price: 105,
				BuyOrders: []OrderPart{
					{"3", 100, 1000, 0, 300},
					{"4", 100, 1000, 0, 400},
				}}, {
				Price: 100,
				BuyOrders: []OrderPart{
					{"5", 100, 1000, 0, 500},
					{"6", 100, 1000, 0, 600},
				}},
			},
		},
		false,
		[]*MergedPriceLevel{{
			price: 100,
			orders: []*OrderPart{
				{"2", 100, 1000, 0, 200},
				{"1", 100, 1000, 0, 100},
				{"4", 100, 1000, 0, 400},
				{"3", 100, 1000, 0, 300},
				{"6", 100, 1000, 0, 600},
				{"5", 100, 1000, 0, 500},
			},
			totalQty: 2100,
		}},
	}, {
		"sellSide_maker_onlyMakerOrders",
		args{
			side:               SELLSIDE,
			height:             100,
			concludedPrice:     100,
			tradePriceLevelIdx: 0,
			levels: []OverLappedLevel{{
				Price: 100,
				SellOrders: []OrderPart{
					{"1", 99, 1000, 0, 100},
				}}, {
				Price: 90,
				SellOrders: []OrderPart{
					{"2", 99, 1000, 0, 200},
					{"3", 99, 1000, 0, 300},
				}},
			},
		},
		true,
		[]*MergedPriceLevel{{
			price: 90,
			orders: []*OrderPart{
				{"3", 99, 1000, 0, 300},
				{"2", 99, 1000, 0, 200},
			},
			totalQty: 500,
		}, {
			price: 100,
			orders: []*OrderPart{
				{"1", 99, 1000, 0, 100},
			},
			totalQty: 100,
		}},
	}, {"buySide_maker_concludedLevelContainsMakerOrders",
		args{
			side:               BUYSIDE,
			height:             100,
			concludedPrice:     100,
			tradePriceLevelIdx: 2,
			levels: []OverLappedLevel{{
				Price: 110,
				BuyOrders: []OrderPart{
					{"1", 99, 1000, 0, 100},
					{"2", 100, 1000, 0, 200},
				}}, {
				Price: 105,
				BuyOrders: []OrderPart{
					{"3", 99, 1000, 0, 300},
					{"4", 100, 1000, 0, 400},
					{"5", 100, 1000, 0, 500},
				}}, {
				Price: 100,
				BuyOrders: []OrderPart{
					{"6", 99, 1000, 0, 600},
					{"7", 100, 1000, 0, 700},
					{"8", 100, 1000, 0, 800},
				}},
			},
		},
		true,
		[]*MergedPriceLevel{{
			price: 110,
			orders: []*OrderPart{
				{"1", 99, 1000, 0, 100},
			},
			totalQty: 100,
		}, {
			price: 105,
			orders: []*OrderPart{
				{"3", 99, 1000, 0, 300},
			},
			totalQty: 300,
		}, {
			price: 100,
			orders: []*OrderPart{
				{"6", 99, 1000, 0, 600},
				{"2", 100, 1000, 0, 200},
				{"5", 100, 1000, 0, 500},
				{"4", 100, 1000, 0, 400},
				{"8", 100, 1000, 0, 800},
				{"7", 100, 1000, 0, 700},
			},
			totalQty: 3200,
		}},
	}, {"buySide_maker_appendConcludedLevel",
		args{
			side:               BUYSIDE,
			height:             100,
			concludedPrice:     100,
			tradePriceLevelIdx: 2,
			levels: []OverLappedLevel{{
				Price: 110,
				BuyOrders: []OrderPart{
					{"1", 99, 1000, 0, 100},
					{"2", 100, 1000, 0, 200},
				}}, {
				Price: 105,
				BuyOrders: []OrderPart{
					{"3", 99, 1000, 0, 300},
					{"4", 100, 1000, 0, 400},
					{"5", 100, 1000, 0, 500},
				}}, {
				Price: 100,
				BuyOrders: []OrderPart{
					{"6", 100, 1000, 0, 600},
					{"7", 100, 1000, 0, 700},
					{"8", 100, 1000, 0, 800},
				}},
			},
		},
		true,
		[]*MergedPriceLevel{{
			price: 110,
			orders: []*OrderPart{
				{"1", 99, 1000, 0, 100},
			},
			totalQty: 100,
		}, {
			price: 105,
			orders: []*OrderPart{
				{"3", 99, 1000, 0, 300},
			},
			totalQty: 300,
		}, {
			price: 100,
			orders: []*OrderPart{
				{"2", 100, 1000, 0, 200},
				{"5", 100, 1000, 0, 500},
				{"4", 100, 1000, 0, 400},
				{"8", 100, 1000, 0, 800},
				{"7", 100, 1000, 0, 700},
				{"6", 100, 1000, 0, 600},
			},
			totalQty: 3200,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsMakerSide, gotMergedLevels := mergeSidePriceLevels(tt.args.side, tt.args.height, tt.args.concludedPrice, tt.args.tradePriceLevelIdx, tt.args.levels)
			if gotIsMakerSide != tt.wantIsMakerSide {
				t.Errorf("mergeSidePriceLevels() gotIsMakerSide = %v, want %v", gotIsMakerSide, tt.wantIsMakerSide)
			}
			assert.EqualValues(t, tt.wantMergedLevels, gotMergedLevels)
		})
	}
}

func Test_createMakerTakerOrders(t *testing.T) {
	type args struct {
		height             int64
		overlapped         []OverLappedLevel
		concludedPrice     int64
		tradePriceLevelIdx int
	}
	tests := []struct {
		name    string
		args    args
		want    *MakerTakerOrders
		wantErr bool
	}{{
		name: "buySideIsMakerSide",
		args: args{
			height:             100,
			concludedPrice:     100,
			tradePriceLevelIdx: 2,
			overlapped: []OverLappedLevel{{
				Price: 120,
				BuyOrders: []OrderPart{
					{"1", 100, 1000, 0, 100},
				},
				SellOrders: []OrderPart{
					{"2", 99, 1000, 0, 200},
					{"4", 100, 1000, 0, 400},
				},
			}, {
				Price: 110,
				BuyOrders: []OrderPart{
					{"3", 99, 1000, 0, 300},
					{"5", 100, 1000, 0, 500},
					{"7", 100, 1000, 0, 700},
				},
				SellOrders: []OrderPart{
					{"6", 99, 1000, 0, 600},
					{"8", 100, 1000, 0, 800},
				},
			}, {
				Price: 100,
				BuyOrders: []OrderPart{
					{"9", 98, 1000, 0, 300},
					{"11", 99, 1000, 0, 0},
					{"13", 100, 1000, 0, 300},
				},
				SellOrders: []OrderPart{
					{"10", 100, 1000, 0, 1000},
					{"12", 100, 1000, 0, 200},
				},
			}, {
				Price: 90,
				BuyOrders: []OrderPart{
					{"15", 99, 1000, 0, 500},
					{"17", 100, 1000, 0, 700},
				},
				SellOrders: []OrderPart{
					{"14", 100, 1000, 0, 400},
					{"16", 100, 1000, 0, 600},
				},
			}},
		},
		want: &MakerTakerOrders{
			isBuySideMaker: true,
			makerSide: MakerSideOrders{
				priceLevels: []*MergedPriceLevel{{
					price: 110,
					orders: []*OrderPart{
						{"3", 99, 1000, 0, 300},
					},
					totalQty: 300,
				}, {
					price: 100,
					orders: []*OrderPart{
						{"9", 98, 1000, 0, 300},
						{"1", 100, 1000, 0, 100},
						{"7", 100, 1000, 0, 700},
						{"5", 100, 1000, 0, 500},
						{"13", 100, 1000, 0, 300},
					},
					totalQty: 1900,
				}},
			},
			takerSide: TakerSideOrders{
				&MergedPriceLevel{
					price: 100,
					orders: []*OrderPart{
						{"16", 100, 1000, 0, 600},
						{"14", 100, 1000, 0, 400},
						{"10", 100, 1000, 0, 1000},
						{"12", 100, 1000, 0, 200},
					},
					totalQty: 2200,
				},
			},
		},
		wantErr: false,
	}, {
		name: "sellSideIsMakerSide",
		args: args{
			height:             100,
			concludedPrice:     110,
			tradePriceLevelIdx: 1,
			overlapped: []OverLappedLevel{{
				Price: 120,
				BuyOrders: []OrderPart{
					{"1", 100, 1000, 0, 2000},
				},
				SellOrders: []OrderPart{
					{"2", 99, 1000, 0, 200},
					{"4", 100, 1000, 0, 400},
				},
			}, {
				Price: 110,
				BuyOrders: []OrderPart{
					{"3", 100, 1000, 0, 300},
					{"5", 100, 1000, 0, 500},
					{"7", 100, 1000, 0, 700},
				},
				SellOrders: []OrderPart{
					{"6", 98, 1000, 0, 600},
					{"8", 99, 1000, 0, 700},
				},
			}, {
				Price: 100,
				BuyOrders: []OrderPart{
					{"9", 98, 1000, 0, 300},
					{"11", 99, 1000, 0, 0},
					{"13", 100, 1000, 0, 300},
				},
				SellOrders: []OrderPart{
					{"10", 100, 1000, 0, 1000},
					{"12", 100, 1000, 0, 200},
				},
			}, {
				Price: 90,
				BuyOrders: []OrderPart{
					{"15", 99, 1000, 0, 500},
					{"17", 100, 1000, 0, 700},
				},
				SellOrders: []OrderPart{
					{"14", 100, 1000, 0, 400},
					{"16", 100, 1000, 0, 600},
				},
			}},
		},
		want: &MakerTakerOrders{
			isBuySideMaker: false,
			makerSide: MakerSideOrders{
				priceLevels: []*MergedPriceLevel{{
					price: 110,
					orders: []*OrderPart{
						{"8", 99, 1000, 0, 700},
						{"6", 98, 1000, 0, 600},
						{"16", 100, 1000, 0, 600},
						{"14", 100, 1000, 0, 400},
						{"10", 100, 1000, 0, 1000},
						{"12", 100, 1000, 0, 200},
					},
					totalQty: 3500,
				}},
			},
			takerSide: TakerSideOrders{
				&MergedPriceLevel{
					price: 110,
					orders: []*OrderPart{
						{"1", 100, 1000, 0, 2000},
						{"7", 100, 1000, 0, 700},
						{"5", 100, 1000, 0, 500},
						{"3", 100, 1000, 0, 300},
					},
					totalQty: 3500,
				},
			},
		},
		wantErr: false,
	}, {
		name: "noMakerOrders",
		args: args{
			height:             100,
			concludedPrice:     110,
			tradePriceLevelIdx: 1,
			overlapped: []OverLappedLevel{{
				Price: 120,
				BuyOrders: []OrderPart{
					{"1", 100, 1000, 0, 2000},
				},
				SellOrders: []OrderPart{
					{"2", 100, 1000, 0, 200},
					{"4", 100, 1000, 0, 400},
				},
			}, {
				Price: 110,
				BuyOrders: []OrderPart{
					{"3", 100, 1000, 0, 300},
					{"5", 100, 1000, 0, 500},
					{"7", 100, 1000, 0, 700},
				},
				SellOrders: []OrderPart{
					{"6", 100, 1000, 0, 600},
					{"8", 100, 1000, 0, 700},
				},
			}, {
				Price: 100,
				BuyOrders: []OrderPart{
					{"9", 100, 1000, 0, 300},
					{"11", 100, 1000, 0, 0},
					{"13", 100, 1000, 0, 300},
				},
				SellOrders: []OrderPart{
					{"10", 100, 1000, 0, 1000},
					{"12", 100, 1000, 0, 200},
				},
			}, {
				Price: 90,
				BuyOrders: []OrderPart{
					{"15", 100, 1000, 0, 500},
					{"17", 100, 1000, 0, 700},
				},
				SellOrders: []OrderPart{
					{"14", 100, 1000, 0, 400},
					{"16", 100, 1000, 0, 600},
				},
			}},
		},
		want: &MakerTakerOrders{
			isBuySideMaker: false,
			makerSide: MakerSideOrders{
				priceLevels: []*MergedPriceLevel{{
					price: 110,
					orders: []*OrderPart{
						{"16", 100, 1000, 0, 600},
						{"14", 100, 1000, 0, 400},
						{"10", 100, 1000, 0, 1000},
						{"12", 100, 1000, 0, 200},
						{"8", 100, 1000, 0, 700},
						{"6", 100, 1000, 0, 600},
					},
					totalQty: 3500,
				}},
			},
			takerSide: TakerSideOrders{
				&MergedPriceLevel{
					price: 110,
					orders: []*OrderPart{
						{"1", 100, 1000, 0, 2000},
						{"7", 100, 1000, 0, 700},
						{"5", 100, 1000, 0, 500},
						{"3", 100, 1000, 0, 300},
					},
					totalQty: 3500,
				},
			},
		},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createMakerTakerOrders(tt.args.height, tt.args.overlapped, tt.args.concludedPrice, tt.args.tradePriceLevelIdx)
			if (err != nil) != tt.wantErr {
				t.Errorf("createMakerTakerOrders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestMatchEng_fillOrdersNew(t *testing.T) {
	assert := assert.New(t)
	// 1. buy side is maker side
	me := NewMatchEng("AAA_BNB", 100, 5, 0.05)
	makerSideOrders := []*OrderPart{
		{"1", 99, 1000, 700, 300},
		{"2", 99, 1000, 900, 100},
		{"3", 99, 1000, 800, 200},
		{"4", 100, 1000, 700, 300},
		{"5", 100, 1000, 200, 100},
	}
	takerSideOrders := []*OrderPart{
		{"6", 100, 1000, 600, 400},
		{"7", 100, 1000, 900, 100},
		{"8", 100, 1000, 700, 300},
		{"9", 100, 1000, 800, 200},
	}
	makerTakerOrders := &MakerTakerOrders{
		isBuySideMaker: true,
		makerSide: MakerSideOrders{
			priceLevels: []*MergedPriceLevel{{
				price:    110,
				orders:   makerSideOrders[:2],
				totalQty: 400,
			}, {
				price:    100,
				orders:   makerSideOrders[2:],
				totalQty: 600,
			}},
		},
		takerSide: TakerSideOrders{
			&MergedPriceLevel{
				price:    100,
				orders:   takerSideOrders,
				totalQty: 1000,
			},
		},
	}
	me.fillOrdersNew(makerTakerOrders, 100)
	assert.Equal([]Trade{
		{"6", 110, 160, 860, 760, "1", SellTaker},
		{"7", 110, 40, 900, 940, "1", SellTaker},
		{"8", 110, 100, 1000, 800, "1", SellTaker},
		{"8", 110, 20, 920, 820, "2", SellTaker},
		{"9", 110, 80, 1000, 880, "2", SellTaker},

		{"6", 100, 200, 1000, 960, "3", SellTaker},
		{"6", 100, 40, 740, 1000, "4", BuySurplus},
		{"7", 100, 60, 800, 1000, "4", BuySurplus},
		{"8", 100, 180, 980, 1000, "4", BuySurplus},
		{"9", 100, 20, 1000, 900, "4", BuySurplus},
		{"9", 100, 100, 300, 1000, "5", BuySurplus},
	}, me.Trades)
	assert.Equal([]*OrderPart{
		{"1", 99, 1000, 1000, 0},
		{"2", 99, 1000, 1000, 0},
		{"3", 99, 1000, 1000, 0},
		{"4", 100, 1000, 1000, 0},
		{"5", 100, 1000, 300, 0},
	}, makerSideOrders)
	assert.Equal([]*OrderPart{
		{"6", 100, 1000, 1000, 0},
		{"7", 100, 1000, 1000, 0},
		{"8", 100, 1000, 1000, 0},
		{"9", 100, 1000, 1000, 0},
	}, takerSideOrders)

	// 2. sell side is maker side
	me = NewMatchEng("AAA_BNB", 100, 5, 0.05)
	makerSideOrders = []*OrderPart{
		{"6", 99, 1000, 600, 400},
		{"7", 99, 1000, 900, 100},
		{"8", 99, 1000, 700, 300},
		{"9", 100, 1000, 800, 200},
	}
	takerSideOrders = []*OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 100, 1000, 900, 100},
		{"3", 100, 1000, 800, 200},
		{"4", 100, 1000, 700, 300},
		{"5", 100, 1000, 200, 100},
	}
	makerTakerOrders = &MakerTakerOrders{
		isBuySideMaker: false,
		makerSide: MakerSideOrders{
			priceLevels: []*MergedPriceLevel{{
				price:    100,
				orders:   makerSideOrders[:2],
				totalQty: 500,
			}, {
				price:    90,
				orders:   makerSideOrders[2:],
				totalQty: 500,
			}},
		},
		takerSide: TakerSideOrders{
			&MergedPriceLevel{
				price:    100,
				orders:   takerSideOrders,
				totalQty: 1000,
			},
		},
	}
	me.fillOrdersNew(makerTakerOrders, -100)
	assert.Equal([]Trade{
		{"6", 100, 150, 850, 750, "1", BuyTaker},
		{"6", 100, 50, 950, 800, "2", BuyTaker},
		{"6", 100, 100, 900, 900, "3", BuyTaker},
		{"6", 100, 100, 800, 1000, "4", BuyTaker},
		{"7", 100, 50, 850, 950, "4", BuyTaker},
		{"7", 100, 50, 250, 1000, "5", BuyTaker},

		{"8", 90, 150, 1000, 850, "1", BuyTaker},
		{"8", 90, 50, 1000, 900, "2", BuyTaker},
		{"8", 90, 100, 1000, 1000, "3", BuyTaker},
		{"9", 90, 150, 1000, 950, "4", SellSurplus},
		{"9", 90, 50, 300, 1000, "5", SellSurplus},
	}, me.Trades)
	assert.Equal([]*OrderPart{
		{"6", 99, 1000, 1000, 0},
		{"7", 99, 1000, 1000, 0},
		{"8", 99, 1000, 1000, 0},
		{"9", 100, 1000, 1000, 0},
	}, makerSideOrders)
	assert.Equal([]*OrderPart{
		{"1", 100, 1000, 1000, 0},
		{"2", 100, 1000, 1000, 0},
		{"3", 100, 1000, 1000, 0},
		{"4", 100, 1000, 1000, 0},
		{"5", 100, 1000, 300, 0},
	}, takerSideOrders)

	// 3. no maker orders
	me = NewMatchEng("AAA_BNB", 100, 5, 0.05)
	makerSideOrders = []*OrderPart{
		{"1", 100, 1000, 700, 300},
		{"2", 100, 1000, 900, 100},
		{"3", 100, 1000, 800, 200},
		{"4", 100, 1000, 700, 300},
		{"5", 100, 1000, 200, 100},
	}
	takerSideOrders = []*OrderPart{
		{"6", 100, 1000, 600, 400},
		{"7", 100, 1000, 900, 100},
		{"8", 100, 1000, 700, 300},
		{"9", 100, 1000, 800, 200},
	}
	makerTakerOrders = &MakerTakerOrders{
		isBuySideMaker: false,
		makerSide: MakerSideOrders{
			priceLevels: []*MergedPriceLevel{{
				price:    100,
				orders:   makerSideOrders,
				totalQty: 1000,
			}},
		},
		takerSide: TakerSideOrders{
			&MergedPriceLevel{
				price:    100,
				orders:   takerSideOrders,
				totalQty: 1000,
			},
		},
	}
	me.fillOrdersNew(makerTakerOrders, 0)
	assert.Equal([]Trade{
		{"1", 100, 300, 900, 1000, "6", Neutral},
		{"2", 100, 100, 1000, 1000, "6", Neutral},
		{"3", 100, 100, 1000, 900, "7", Neutral},
		{"3", 100, 100, 800, 1000, "8", Neutral},
		{"4", 100, 200, 1000, 900, "8", Neutral},
		{"4", 100, 100, 900, 1000, "9", Neutral},
		{"5", 100, 100, 1000, 300, "9", Neutral},
	}, me.Trades)
	assert.Equal([]*OrderPart{
		{"1", 100, 1000, 1000, 0},
		{"2", 100, 1000, 1000, 0},
		{"3", 100, 1000, 1000, 0},
		{"4", 100, 1000, 1000, 0},
		{"5", 100, 1000, 300, 0},
	}, makerSideOrders)
	assert.Equal([]*OrderPart{
		{"6", 100, 1000, 1000, 0},
		{"7", 100, 1000, 1000, 0},
		{"8", 100, 1000, 1000, 0},
		{"9", 100, 1000, 1000, 0},
	}, takerSideOrders)

}

func TestMatchEng_Match(t *testing.T) {
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

	assert.True(me.Match(100))
	assert.Equal(4, len(me.overLappedLevel))
	assert.Equal(int64(100), me.LastTradePrice)
	assert.Equal([]Trade{
		{"7", 100, 25, 25, 25, "12", BuyTaker},
		{"3", 100, 10, 35, 10, "12", BuyTaker},
		{"1", 100, 5, 40, 5, "12", BuyTaker},
		{"15", 100, 40, 80, 40, "12", SellSurplus},
		{"17", 100, 20, 100, 20, "12", SellSurplus},
		{"13", 100, 10, 110, 10, "12", SellSurplus},
		{"11", 100, 20, 20, 20, "16", SellSurplus},
		{"11", 100, 10, 10, 30, "14", SellSurplus},
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
	me.Book.InsertOrder("1", SELLSIDE, 90, 100, 25)
	me.Book.InsertOrder("3", SELLSIDE, 90, 100, 25)
	me.Book.InsertOrder("5", SELLSIDE, 90, 100, 25)
	me.Book.InsertOrder("2", BUYSIDE, 92, 90, 5)
	me.Book.InsertOrder("4", BUYSIDE, 93, 80, 30)
	me.Book.InsertOrder("11", SELLSIDE, 100, 90, 30)
	me.Book.InsertOrder("13", SELLSIDE, 100, 80, 10)
	me.Book.InsertOrder("15", SELLSIDE, 100, 80, 40)
	me.Book.InsertOrder("17", SELLSIDE, 100, 80, 20)
	me.Book.InsertOrder("12", BUYSIDE, 100, 110, 140)

	assert.True(me.Match(100))
	assert.Equal(4, len(me.overLappedLevel))
	assert.Equal(int64(104), me.LastTradePrice)
	assert.Equal([]Trade{
		{"1", 100, 20, 20, 20, "12", BuyTaker},
		{"3", 100, 10, 30, 10, "12", BuyTaker},
		{"5", 100, 10, 40, 10, "12", BuyTaker},
		{"15", 104, 40, 80, 40, "12", SellSurplus},
		{"17", 104, 20, 100, 20, "12", SellSurplus},
		{"13", 104, 10, 110, 10, "12", SellSurplus},
		{"11", 104, 30, 140, 30, "12", SellSurplus},
	}, me.Trades)
	me.DropFilledOrder()
	buys, sells = me.Book.GetAllLevels()
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
			{"1", 90, 25, 20, 5},
			{"3", 90, 25, 10, 15},
			{"5", 90, 25, 10, 15},
		},
	}}, sells)
}
