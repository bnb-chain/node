package matcheng

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	bt "github.com/google/btree"
	"github.com/stretchr/testify/assert"
)

func Test_compareBuy(t *testing.T) {
	type args struct {
		p1 int64
		p2 int64
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"P1Bigger1", args{501, 500}, 1},
		{"P1Bigger2", args{1005, 1000}, 1},
		{"P1Bigger3", args{-111, -112}, 1},
		{"P1Bigger4", args{1, 0}, 1},
		{"Equal1", args{0, 0}, 0},
		{"Equal2", args{10, 10}, 0},
		{"Equal3", args{-1, -1}, 0},
		{"Equal4", args{5581234, 5581234}, 0},
		{"Equal5", args{55812345607, 55812345607}, 0},
		{"P2Bigger1", args{50, 51}, -1},
		{"P2Bigger2", args{5, 1005}, -1},
		{"P2Bigger3", args{-11, -10}, -1},
		{"P2Bigger4", args{0, 1}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareBuy(tt.args.p1, tt.args.p2); got != tt.want {
				t.Errorf("compareBuy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriceLevel_addOrder(t *testing.T) {
	type fields struct {
		Price  int64
		orders []OrderPart
	}
	type args struct {
		id   string
		time int64
		qty  int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{"AddedOrder", fields{1000, make([]OrderPart, 0, 1)}, args{"12345", 2354, 10005}, 1, false},
		{"Duplicated", fields{1000, []OrderPart{{"12345", 0, 1555, 0, 0}}}, args{"12345", 2354, 10005}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &PriceLevel{
				Price:  tt.fields.Price,
				orders: tt.fields.orders,
			}
			got, err := l.addOrder(tt.args.id, tt.args.time, tt.args.qty)
			if (err != nil) != tt.wantErr {
				t.Errorf("PriceLevel.addOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.name == "AddedOrder" {
				if l.orders[0].id != tt.args.id ||
					l.orders[0].qty != tt.args.qty ||
					l.orders[0].time != tt.args.time {
					t.Error("order is not inserted into PriceLevel")
				}
			}
			if got != tt.want {
				t.Errorf("PriceLevel.addOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriceLevel_removeOrder(t *testing.T) {
	type fields struct {
		Price  int64
		orders []OrderPart
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    OrderPart
		want1   int
		wantErr bool
	}{
		{"NotExist1", fields{1000, []OrderPart{{"12345", 0, 1555, 0, 0}}}, args{"12346"}, OrderPart{}, 0, true},
		{"NotExist2", fields{1000, []OrderPart{}}, args{"12346"}, OrderPart{}, 0, true},
		{"Delete1", fields{1000, []OrderPart{{"12345", 0, 1555, 0, 0}, {"12346", 0, 1556, 0, 0},
			{"12347", 0, 1557, 0, 0}}}, args{"12345"}, OrderPart{"12345", 0, 1555, 0, 0}, 2, false},
		{"Delete2", fields{1000, []OrderPart{{"12345", 0, 1555, 0, 0}, {"12346", 0, 1556, 0, 0},
			{"12347", 0, 1557, 0, 0}}}, args{"12347"}, OrderPart{"12347", 0, 1557, 0, 0}, 2, false},
		{"Delete3", fields{1000, []OrderPart{{"12345", 0, 1555, 0, 0}, {"12346", 0, 1556, 0, 0},
			{"12347", 0, 1557, 0, 0}}}, args{"12346"}, OrderPart{"12346", 0, 1556, 0, 0}, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &PriceLevel{
				Price:  tt.fields.Price,
				orders: tt.fields.orders,
			}
			got, got1, err := l.removeOrder(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("PriceLevel.removeOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.name == "Delete2" {
				if len(l.orders) != 2 || l.orders[0].id != "12345" || l.orders[1].id != "12346" {
					t.Error("RemoveOrder failed to remove correct id")
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PriceLevel.removeOrder() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("PriceLevel.removeOrder() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_mergeLevels(t *testing.T) {
	type args struct {
		buyLevels  []PriceLevel
		sellLevels []PriceLevel
		overlapped *[]OverLappedLevel
	}
	overLapped := make([]OverLappedLevel, 2)
	tests := []struct {
		name string
		args args
	}{
		{"ClearOverlapped", args{nil, nil, &overLapped}},
		{"OneSide1", args{[]PriceLevel{{Price: 1200}, {Price: 1010}, {Price: 1000}}, nil, &overLapped}},
		{"OneSide2", args{nil, []PriceLevel{{Price: 1000}, {Price: 1010}, {Price: 1200}}, &overLapped}},
		{"OneOnEachSide", args{[]PriceLevel{{Price: 1010}}, []PriceLevel{{Price: 1000}}, &overLapped}},
		{"NormalMerge", args{[]PriceLevel{{Price: 1041, orders: make([]OrderPart, 3)}, {Price: 1030}}, []PriceLevel{{Price: 1020}, {Price: 1032, orders: make([]OrderPart, 4)}}, &overLapped}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeLevels(tt.args.buyLevels, tt.args.sellLevels, tt.args.overlapped)
			if strings.Contains(tt.name, "Clear") {
				if len(overLapped) != 0 {
					t.Error("mergeLevel doesn't clean the input overlapped parameter.")
				}
			}
			if strings.Contains(tt.name, "OneSide") {
				if len(overLapped) != 3 || overLapped[0].Price != 1200 {
					t.Error("mergeLevel failed to merage one side")
				}
			}
			if strings.Contains(tt.name, "OneOnEach") {
				if len(overLapped) != 2 || overLapped[0].Price != 1010 || overLapped[1].Price != 1000 {
					t.Errorf("mergeLevel failed to merage one on each side %v, %v, %v",
						len(overLapped), overLapped[0].Price, overLapped[1].Price)
				}
			}
			if strings.Contains(tt.name, "Normal") {
				if len(overLapped) != 4 || overLapped[0].Price != 1041 || overLapped[1].Price != 1032 {
					t.Error("mergeLevel failed to merage with correct number")
				}
				if len(overLapped[0].BuyOrders) != 3 || len(overLapped[0].SellOrders) != 0 ||
					len(overLapped[1].BuyOrders) != 0 || len(overLapped[1].SellOrders) != 4 ||
					len(overLapped[3].BuyOrders) != 0 || len(overLapped[3].SellOrders) != 0 {
					t.Error("mergeLevel failed to merge with correct orders")
				}
			}

		})
	}
}

func TestOrderBookOnULList_InsertOrder(t *testing.T) {
	type fields struct {
		buyQueue  *ULList
		sellQueue *ULList
	}
	type args struct {
		id    string
		side  int
		time  int64
		price int64
		qty   int64
	}

	samePrice := func() *OrderBookOnULList {
		l := NewOrderBookOnULList(16, 4)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123457", BUYSIDE, 10001, 1000, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	newPrice := func() *OrderBookOnULList {
		l := NewOrderBookOnULList(16, 4)
		l.InsertOrder("123459", BUYSIDE, 10002, 1005, 10000)
		l.InsertOrder("123459", BUYSIDE, 10002, 995, 10000)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	newPrice2 := func() *OrderBookOnULList {
		l := NewOrderBookOnULList(16, 4)
		l.InsertOrder("123459", BUYSIDE, 10002, 1005, 10000)
		l.InsertOrder("123459", BUYSIDE, 10002, 995, 10000)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123457", BUYSIDE, 10001, 1007, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		l.InsertOrder("123460", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	newPrice3 := func() *OrderBookOnULList {
		l := NewOrderBookOnULList(5, 2)
		l.InsertOrder("123459", BUYSIDE, 10002, 1005, 10000)
		l.InsertOrder("123459", BUYSIDE, 10002, 995, 10000)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123457", BUYSIDE, 10001, 1007, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		l.InsertOrder("123460", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *PriceLevel
		wantErr bool
	}{
		{"Sanity", fields{NewULList(4096, 16, compareBuy), NewULList(4096, 16, compareSell)},
			args{"123456", BUYSIDE, 10000, 1000, 10000}, &PriceLevel{1000, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"SamePrice", fields{samePrice().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1000, 10000}, &PriceLevel{1000, []OrderPart{{"123455", 10000, 10000, 0, 0},
				{"123457", 10001, 10000, 0, 0}, {"123458", 10002, 10000, 0, 0}, {"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPrice1", fields{newPrice().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1010, 10000}, &PriceLevel{1010, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPrice2", fields{newPrice().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 990, 10000}, &PriceLevel{990, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit1", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1010, 10000}, &PriceLevel{1010, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit2", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 990, 10000}, &PriceLevel{990, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit3", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1000, 10000}, &PriceLevel{1000, []OrderPart{{"123455", 10000, 10000, 0, 0},
				{"123458", 10002, 10000, 0, 0}, {"123460", 10002, 10000, 0, 0}, {"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit4", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1004, 10000}, &PriceLevel{1004, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit5", fields{newPrice3().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1006, 10000}, &PriceLevel{1006, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBookOnULList{
				buyQueue:  tt.fields.buyQueue,
				sellQueue: tt.fields.sellQueue,
			}
			got, err := ob.InsertOrder(tt.args.id, tt.args.side, tt.args.time, tt.args.price, tt.args.qty)
			//t.Logf("after insert:%s", ob)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderBookOnULList.InsertOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrderBookOnULList.InsertOrder() = %v, want %v", got, tt.want)
			}
			switch tt.name {

			case "SamePrice":

				if len(ob.buyQueue.begin.elements) != 1 ||
					len(ob.buyQueue.begin.elements[0].orders) != 4 ||
					ob.buyQueue.begin.elements[0].orders[0].id != "123455" ||
					ob.buyQueue.begin.elements[0].orders[3].id != "123456" {
					t.Error("SamePrice doesn't work")
				}
			case "NewPrice1":
				if ob.buyQueue.String() != "Bucket 0{1010->[123456 10000 10000,]1005->[123459 10002 10000,]1000->[123455 10000 10000,123458 10002 10000,]995->[123459 10002 10000,]}," {
					t.Errorf("NewPrice1 insert failure:%v", ob.buyQueue)
				}
			case "NewPrice2":
				if ob.buyQueue.String() != "Bucket 0{1005->[123459 10002 10000,]1000->[123455 10000 10000,123458 10002 10000,]995->[123459 10002 10000,]990->[123456 10000 10000,]}," {
					t.Errorf("NewPrice2 insert failure:%v", ob.buyQueue)
				}
			case "NewPriceSplit1":
				if ob.buyQueue.String() != "Bucket 0{1010->[123456 10000 10000,]},Bucket 1{1007->[123457 10001 10000,]1005->[123459 10002 10000,]1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,]995->[123459 10002 10000,]}," {
					t.Errorf("NewPriceSplit1 insert failure:%s", ob.buyQueue)
				}
			case "NewPriceSplit2":
				if ob.buyQueue.String() != "Bucket 0{1007->[123457 10001 10000,]1005->[123459 10002 10000,]1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,]995->[123459 10002 10000,]},Bucket 1{990->[123456 10000 10000,]}," {
					t.Errorf("NewPriceSplit1 insert failure:%s", ob.buyQueue)
				}
			case "NewPriceSplit3":
				if ob.buyQueue.String() != "Bucket 0{1007->[123457 10001 10000,]1005->[123459 10002 10000,]1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,123456 10000 10000,]995->[123459 10002 10000,]}," {
					t.Errorf("NewPriceSplit1 insert failure:%s", ob.buyQueue)
				}
			case "NewPriceSplit4":
				if ob.buyQueue.String() != "Bucket 0{1007->[123457 10001 10000,]1005->[123459 10002 10000,]},Bucket 1{1004->[123456 10000 10000,]1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,]995->[123459 10002 10000,]}," {
					t.Errorf("NewPriceSplit1 insert failure:%s", ob.buyQueue)
				}
			case "NewPriceSplit5":
				if ob.buyQueue.String() != "Bucket 0{1007->[123457 10001 10000,]},Bucket 1{1006->[123456 10000 10000,]1005->[123459 10002 10000,]},Bucket 2{1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,]995->[123459 10002 10000,]}," {
					t.Errorf("NewPriceSplit1 insert failure:%s", ob.buyQueue)
				}
			}
		})
	}
}

func TestOrderBookOnBTree_InsertOrder(t *testing.T) {
	type fields struct {
		buyQueue  *bt.BTree
		sellQueue *bt.BTree
	}
	type args struct {
		id    string
		side  int
		time  int64
		price int64
		qty   int64
	}

	samePrice := func() *OrderBookOnBTree {
		l := NewOrderBookOnBTree(8)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123457", BUYSIDE, 10001, 1000, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	newPrice := func() *OrderBookOnBTree {
		l := NewOrderBookOnBTree(8)
		l.InsertOrder("123459", BUYSIDE, 10002, 1005, 10000)
		l.InsertOrder("123459", BUYSIDE, 10002, 995, 10000)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	newPrice2 := func() *OrderBookOnBTree {
		l := NewOrderBookOnBTree(8)
		l.InsertOrder("123459", BUYSIDE, 10002, 1005, 10000)
		l.InsertOrder("123459", BUYSIDE, 10002, 995, 10000)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123457", BUYSIDE, 10001, 1007, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		l.InsertOrder("123460", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	newPrice3 := func() *OrderBookOnBTree {
		l := NewOrderBookOnBTree(8)
		l.InsertOrder("123459", BUYSIDE, 10002, 1005, 10000)
		l.InsertOrder("123459", BUYSIDE, 10002, 995, 10000)
		l.InsertOrder("123455", BUYSIDE, 10000, 1000, 10000)
		l.InsertOrder("123457", BUYSIDE, 10001, 1007, 10000)
		l.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
		l.InsertOrder("123460", BUYSIDE, 10002, 1000, 10000)
		return l
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *PriceLevel
		wantErr bool
	}{
		{"Sanity", fields{bt.New(8), bt.New(8)},
			args{"123456", BUYSIDE, 10000, 1000, 10000}, &PriceLevel{1000, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"SamePrice", fields{samePrice().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1000, 10000}, &PriceLevel{1000, []OrderPart{{"123455", 10000, 10000, 0, 0},
				{"123457", 10001, 10000, 0, 0}, {"123458", 10002, 10000, 0, 0}, {"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPrice1", fields{newPrice().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1010, 10000}, &PriceLevel{1010, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPrice2", fields{newPrice().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 990, 10000}, &PriceLevel{990, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit1", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1010, 10000}, &PriceLevel{1010, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit2", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 990, 10000}, &PriceLevel{990, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit3", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1000, 10000}, &PriceLevel{1000, []OrderPart{{"123455", 10000, 10000, 0, 0},
				{"123458", 10002, 10000, 0, 0}, {"123460", 10002, 10000, 0, 0}, {"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit4", fields{newPrice2().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1004, 10000}, &PriceLevel{1004, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
		{"NewPriceSplit5", fields{newPrice3().buyQueue, nil},
			args{"123456", BUYSIDE, 10000, 1006, 10000}, &PriceLevel{1006, []OrderPart{{"123456", 10000, 10000, 0, 0}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBookOnBTree{
				buyQueue:  tt.fields.buyQueue,
				sellQueue: tt.fields.sellQueue,
			}

			got, err := ob.InsertOrder(tt.args.id, tt.args.side, tt.args.time, tt.args.price, tt.args.qty)
			t.Log(printOrderQueueString(ob.buyQueue, BUYSIDE))
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderBookOnBTree.InsertOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrderBookOnBTree.InsertOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderBookOnULList_RemoveOrder(t *testing.T) {
	assert := assert.New(t)
	samePrice := NewOrderBookOnULList(16, 4)
	samePrice.InsertOrder("123456", BUYSIDE, 10000, 1000, 10000)
	samePrice.InsertOrder("123457", BUYSIDE, 10001, 1000, 10000)
	samePrice.InsertOrder("123458", BUYSIDE, 10002, 1000, 10000)
	ord, err := samePrice.RemoveOrder("123457", BUYSIDE, 1000)
	assert.Equal(ord, OrderPart{"123457", 10001, 10000, 0, 0}, "Failed to remove middle order from multiple orders at the same price")
	assert.Nil(err)
	ord, err = samePrice.RemoveOrder("123456", BUYSIDE, 1000)
	assert.Equal(ord, OrderPart{"123456", 10000, 10000, 0, 0}, "Failed to remove head order from multiple orders at the same price")
	assert.Nil(err)
	ord, err = samePrice.RemoveOrder("123458", BUYSIDE, 1000)
	assert.Equal(ord, OrderPart{"123458", 10002, 10000, 0, 0}, "Failed to remove last order at the same price")
	assert.Nil(err)

	l := NewOrderBookOnULList(7, 2)
	l.InsertOrder("123459", SELLSIDE, 10002, 1005, 10000)
	l.InsertOrder("123459", SELLSIDE, 10002, 995, 10000)
	l.InsertOrder("123455", SELLSIDE, 10000, 1000, 10000)
	l.InsertOrder("123457", SELLSIDE, 10001, 1007, 10000)
	l.InsertOrder("123458", SELLSIDE, 10002, 1000, 10000)
	l.InsertOrder("123460", SELLSIDE, 10002, 1000, 10000)
	ord, err = l.RemoveOrder("123457", SELLSIDE, 1007)
	assert.Equal(ord, OrderPart{"123457", 10001, 10000, 0, 0}, "Failed to remove last order level")
	assert.Equal("Bucket 0{995->[123459 10002 10000,]},Bucket 1{1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,]1005->[123459 10002 10000,]},",
		l.sellQueue.String(), "Level at 1007 should be removed.")
	ord, err = l.RemoveOrder("123459", SELLSIDE, 995)
	assert.Equal(ord, OrderPart{"123459", 10002, 10000, 0, 0}, "Failed to remove 1st order level")
	assert.Equal("Bucket 0{1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,]1005->[123459 10002 10000,]},",
		l.sellQueue.String(), "Level at 995 should be removed.")
	ord, err = l.RemoveOrder("123459", SELLSIDE, 1005)
	assert.Equal(ord, OrderPart{"123459", 10002, 10000, 0, 0}, "Failed to remove last price")
	assert.Equal("Bucket 0{1000->[123455 10000 10000,123458 10002 10000,123460 10002 10000,]},",
		l.sellQueue.String(), "Level at 1005 should be removed.")
	ord, err = l.RemoveOrder("123455", SELLSIDE, 1000)
	assert.Equal(ord, OrderPart{"123455", 10000, 10000, 0, 0}, "Failed to remove 1st order at the same price")
	assert.Equal("Bucket 0{1000->[123458 10002 10000,123460 10002 10000,]},",
		l.sellQueue.String(), "Level at 1000 should remain.")
	ord, err = l.RemoveOrder("123460", SELLSIDE, 1000)
	assert.Equal(ord, OrderPart{"123460", 10002, 10000, 0, 0}, "Failed to remove last order")
	assert.Equal("Bucket 0{1000->[123458 10002 10000,]},",
		l.sellQueue.String(), "Level at 1000 should remain.")
	ord, err = l.RemoveOrder("123458", SELLSIDE, 1000)
	assert.Equal(ord, OrderPart{"123458", 10002, 10000, 0, 0}, "Failed to remove last order")
	assert.Equal("",
		l.sellQueue.String(), "Level at 1000 should be removed.")
}

func TestOrderBookOnBTree_RemoveOrder(t *testing.T) {
	assert := assert.New(t)
	samePrice := NewOrderBookOnBTree(8)
	samePrice.InsertOrder("123456", BUYSIDE, 10000, 1000, 1000)
	samePrice.InsertOrder("123457", BUYSIDE, 10001, 1000, 1000)
	samePrice.InsertOrder("123458", BUYSIDE, 10002, 1000, 1000)
	ord, err := samePrice.RemoveOrder("123457", BUYSIDE, 1000)
	assert.Equal(ord, OrderPart{"123457", 10001, 1000, 0, 0}, "Failed to remove middle order from multiple orders at the same price")
	assert.Nil(err)
	ord, err = samePrice.RemoveOrder("123456", BUYSIDE, 1000)
	assert.Equal(ord, OrderPart{"123456", 10000, 1000, 0, 0}, "Failed to remove head order from multiple orders at the same price")
	assert.Nil(err)
	ord, err = samePrice.RemoveOrder("123458", BUYSIDE, 1000)
	assert.Equal(ord, OrderPart{"123458", 10002, 1000, 0, 0}, "Failed to remove last order at the same price")
	assert.Nil(err)

	l := NewOrderBookOnBTree(8)
	l.InsertOrder("123459", SELLSIDE, 10002, 1005, 1000)
	l.InsertOrder("123459", SELLSIDE, 10002, 995, 1000)
	l.InsertOrder("123455", SELLSIDE, 10000, 1000, 1000)
	l.InsertOrder("123457", SELLSIDE, 10001, 1007, 1000)
	l.InsertOrder("123458", SELLSIDE, 10002, 1000, 1000)
	l.InsertOrder("123460", SELLSIDE, 10002, 1000, 1000)
	ord, err = l.RemoveOrder("123457", SELLSIDE, 1007)
	assert.Equal(ord, OrderPart{"123457", 10001, 1000, 0, 0}, "Failed to remove last order level")
	assert.Equal("995->[[{123459 10002 1000 0 0}]], 1000->[[{123455 10000 1000 0 0} {123458 10002 1000 0 0} {123460 10002 1000 0 0}]], 1005->[[{123459 10002 1000 0 0}]], ",
		printOrderQueueString(l.sellQueue, SELLSIDE), "Level at 1007 should be removed.")
	ord, err = l.RemoveOrder("123459", SELLSIDE, 995)
	assert.Equal(ord, OrderPart{"123459", 10002, 1000, 0, 0}, "Failed to remove 1st order level")
	assert.Equal("1000->[[{123455 10000 1000 0 0} {123458 10002 1000 0 0} {123460 10002 1000 0 0}]], 1005->[[{123459 10002 1000 0 0}]], ",
		printOrderQueueString(l.sellQueue, SELLSIDE), "Level at 995 should be removed.")
	ord, err = l.RemoveOrder("123459", SELLSIDE, 1005)
	assert.Equal(ord, OrderPart{"123459", 10002, 1000, 0, 0}, "Failed to remove last price")
	assert.Equal("1000->[[{123455 10000 1000 0 0} {123458 10002 1000 0 0} {123460 10002 1000 0 0}]], ",
		printOrderQueueString(l.sellQueue, SELLSIDE), "Level at 1005 should be removed.")
	ord, err = l.RemoveOrder("123455", SELLSIDE, 1000)
	assert.Equal(ord, OrderPart{"123455", 10000, 1000, 0, 0}, "Failed to remove 1st order at the same price")
	assert.Equal("1000->[[{123458 10002 1000 0 0} {123460 10002 1000 0 0}]], ",
		printOrderQueueString(l.sellQueue, SELLSIDE), "Level at 1000 should remain.")
	ord, err = l.RemoveOrder("123460", SELLSIDE, 1000)
	assert.Equal(ord, OrderPart{"123460", 10002, 1000, 0, 0}, "Failed to remove last order")
	assert.Equal("1000->[[{123458 10002 1000 0 0}]], ",
		printOrderQueueString(l.sellQueue, SELLSIDE), "Level at 1000 remain.")
	ord, err = l.RemoveOrder("123458", SELLSIDE, 1000)
	assert.Equal(ord, OrderPart{"123458", 10002, 1000, 0, 0}, "Failed to remove last order")
	assert.Equal("",
		printOrderQueueString(l.sellQueue, SELLSIDE), "Level at 1000 be removed.")
}

func TestOrderBookOnULList_GetOverlappedRange(t *testing.T) {
	overlap := make([]OverLappedLevel, 4)
	buyBuf := make([]PriceLevel, 16)
	sellBuf := make([]PriceLevel, 16)
	assert := assert.New(t)
	l := NewOrderBookOnULList(7, 3)
	l.InsertOrder("123451", SELLSIDE, 10000, 9950, 1000)
	l.InsertOrder("123452", SELLSIDE, 10000, 9955, 1000)
	l.InsertOrder("123453", SELLSIDE, 10001, 10000, 1000)
	l.InsertOrder("123454", SELLSIDE, 10002, 10000, 1000)
	l.InsertOrder("123455", SELLSIDE, 10002, 10010, 1000)
	l.InsertOrder("123456", SELLSIDE, 10002, 10020, 1000)
	l.InsertOrder("123457", SELLSIDE, 10003, 10020, 1000)
	l.InsertOrder("123458", SELLSIDE, 10003, 10021, 1000)
	l.InsertOrder("123459", SELLSIDE, 10004, 10022, 1000)
	l.InsertOrder("123460", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123461", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123462", SELLSIDE, 10005, 10032, 1000)
	l.InsertOrder("123463", SELLSIDE, 10005, 10033, 1000)
	t.Log(l.sellQueue)
	assert.Equal(l.sellQueue.capacity, 14, "Capacity expansion")
	assert.Equal(0, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf))

	l.InsertOrder("223451", BUYSIDE, 10000, 9950, 1000)
	l.InsertOrder("223452", BUYSIDE, 10000, 9955, 1000)
	l.InsertOrder("223453", BUYSIDE, 10001, 10000, 1000)
	l.InsertOrder("223454", BUYSIDE, 10002, 10000, 1000)
	l.InsertOrder("223455", BUYSIDE, 10002, 10010, 1000)
	l.InsertOrder("223456", BUYSIDE, 10002, 10020, 1000)
	l.InsertOrder("223457", BUYSIDE, 10003, 10020, 1000)
	l.InsertOrder("223458", BUYSIDE, 10003, 10021, 1000)
	l.InsertOrder("223459", BUYSIDE, 10004, 10022, 1000)
	l.InsertOrder("223460", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223461", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223462", BUYSIDE, 10005, 10032, 1000)
	l.InsertOrder("223463", BUYSIDE, 10005, 10033, 1000)
	t.Log(l.buyQueue)
	assert.Equal(l.buyQueue.capacity, 14, "Capacity expansion")

	assert.Equal(10, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf), "10 price overlap")
	t.Log(overlap)
	var j int
	for b := l.buyQueue.begin; b != l.buyQueue.dend; b = b.next {
		for _, p := range b.elements {
			assert.Equal(p.Price, overlap[j].Price, "overlapped Price equal")
			assert.Equal(2*len(p.orders), len(overlap[j].BuyOrders)+len(overlap[j].SellOrders), "order number equal")
			j++
		}
	}
	l.buyQueue = NewULList(7, 3, compareBuy)
	l.InsertOrder("223451", BUYSIDE, 10000, 9750, 1000)
	l.InsertOrder("223452", BUYSIDE, 10000, 9855, 1000)
	l.InsertOrder("223453", BUYSIDE, 10001, 9860, 1000)
	l.InsertOrder("223454", BUYSIDE, 10002, 10001, 1000)
	l.InsertOrder("223455", BUYSIDE, 10002, 10011, 1000)
	l.InsertOrder("223456", BUYSIDE, 10002, 10021, 1000)
	l.InsertOrder("223457", BUYSIDE, 10003, 10021, 1000)
	l.InsertOrder("223458", BUYSIDE, 10003, 10024, 1000)
	l.InsertOrder("223459", BUYSIDE, 10004, 10025, 1000)
	l.InsertOrder("223460", BUYSIDE, 10005, 10031, 1000)
	l.InsertOrder("223461", BUYSIDE, 10005, 10031, 1000)
	l.InsertOrder("223462", BUYSIDE, 10005, 10034, 1000)
	l.InsertOrder("223463", BUYSIDE, 10005, 10035, 1000)
	assert.Equal(17, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf), "10 price overlap")
	t.Log(overlap)
	type PriceOrd struct {
		price int64
		ordNo int
	}
	result := []PriceOrd{{10035, 1}, {10034, 1}, {10033, 1}, {10032, 1}, {10031, 2},
		{10030, 2}, {10025, 1}, {10024, 1}, {10022, 1}, {10021, 3}, {10020, 2},
		{10011, 1}, {10010, 1}, {10001, 1}, {10000, 2}, {9955, 1}, {9950, 1}}

	for j, o := range overlap {
		assert.Equal(o.Price, result[j].price, "overlapped Price equal")
		assert.Equal(len(o.BuyOrders)+len(o.SellOrders), result[j].ordNo, "order number equal")

	}
	l.buyQueue = NewULList(7, 3, compareBuy)
	l.InsertOrder("223451", BUYSIDE, 10000, 9950, 1000)
	l.InsertOrder("223452", BUYSIDE, 10000, 9955, 1000)
	l.InsertOrder("223453", BUYSIDE, 10001, 10000, 1000)
	l.InsertOrder("223454", BUYSIDE, 10002, 10000, 1000)
	l.InsertOrder("223455", BUYSIDE, 10002, 10010, 1000)
	l.InsertOrder("223456", BUYSIDE, 10002, 10020, 1000)
	l.InsertOrder("223457", BUYSIDE, 10003, 10020, 1000)
	l.InsertOrder("223458", BUYSIDE, 10003, 10021, 1000)
	l.InsertOrder("223459", BUYSIDE, 10004, 10022, 1000)
	l.InsertOrder("223460", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223461", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223462", BUYSIDE, 10005, 10032, 1000)
	l.InsertOrder("223463", BUYSIDE, 10005, 10033, 1000)
	l.sellQueue = NewULList(7, 3, compareSell)
	assert.Equal(0, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf))
	l.InsertOrder("123451", SELLSIDE, 10000, 9750, 1000)
	l.InsertOrder("123452", SELLSIDE, 10000, 9855, 1000)
	l.InsertOrder("123453", SELLSIDE, 10001, 9860, 1000)
	l.InsertOrder("123454", SELLSIDE, 10002, 10000, 1000)
	l.InsertOrder("123455", SELLSIDE, 10002, 10010, 1000)
	l.InsertOrder("123456", SELLSIDE, 10002, 10020, 1000)
	l.InsertOrder("123457", SELLSIDE, 10003, 10020, 1000)
	l.InsertOrder("123458", SELLSIDE, 10003, 10021, 1000)
	l.InsertOrder("123459", SELLSIDE, 10004, 10022, 1000)
	l.InsertOrder("123460", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123461", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123462", SELLSIDE, 10005, 10132, 1000)
	l.InsertOrder("123463", SELLSIDE, 10005, 10133, 1000)
	assert.Equal(13, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf), "10 price overlap")
	t.Log(overlap)
	j = 0
	for b := l.buyQueue.begin; b != l.buyQueue.dend; b = b.next {
		for _, p := range b.elements {
			assert.Equal(p.Price, overlap[j].Price, "overlapped Price equal")
			j++
		}
	}
	assert.Equal(int64(9860), overlap[j].Price)
	assert.Equal(int64(9855), overlap[j+1].Price)
	assert.Equal(int64(9750), overlap[j+2].Price)
}

func TestOrderBookOnBTree_GetOverlappedRange(t *testing.T) {
	assert := assert.New(t)
	l := NewOrderBookOnBTree(8)
	l.InsertOrder("123451", SELLSIDE, 10000, 9950, 1000)
	l.InsertOrder("123452", SELLSIDE, 10000, 9955, 1000)
	l.InsertOrder("123453", SELLSIDE, 10001, 10000, 1000)
	l.InsertOrder("123454", SELLSIDE, 10002, 10000, 1000)
	l.InsertOrder("123455", SELLSIDE, 10002, 10010, 1000)
	l.InsertOrder("123456", SELLSIDE, 10002, 10020, 1000)
	l.InsertOrder("123457", SELLSIDE, 10003, 10020, 1000)
	l.InsertOrder("123458", SELLSIDE, 10003, 10021, 1000)
	l.InsertOrder("123459", SELLSIDE, 10004, 10022, 1000)
	l.InsertOrder("123460", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123461", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123462", SELLSIDE, 10005, 10032, 1000)
	l.InsertOrder("123463", SELLSIDE, 10005, 10033, 1000)
	t.Log(printOrderQueueString(l.sellQueue, SELLSIDE))
	assert.Equal(10, l.sellQueue.Len(), "Sell Queue Len")

	l.InsertOrder("223451", BUYSIDE, 10000, 9950, 1000)
	l.InsertOrder("223452", BUYSIDE, 10000, 9955, 1000)
	l.InsertOrder("223453", BUYSIDE, 10001, 10000, 1000)
	l.InsertOrder("223454", BUYSIDE, 10002, 10000, 1000)
	l.InsertOrder("223455", BUYSIDE, 10002, 10010, 1000)
	l.InsertOrder("223456", BUYSIDE, 10002, 10020, 1000)
	l.InsertOrder("223457", BUYSIDE, 10003, 10020, 1000)
	l.InsertOrder("223458", BUYSIDE, 10003, 10021, 1000)
	l.InsertOrder("223459", BUYSIDE, 10004, 10022, 1000)
	l.InsertOrder("223460", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223461", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223462", BUYSIDE, 10005, 10032, 1000)
	l.InsertOrder("223463", BUYSIDE, 10005, 10033, 1000)
	t.Log(printOrderQueueString(l.buyQueue, BUYSIDE))
	assert.Equal(10, l.buyQueue.Len(), "Buy Queue Len")
	overlap := make([]OverLappedLevel, 4)
	buyBuf := make([]PriceLevel, 16)
	sellBuf := make([]PriceLevel, 16)
	assert.Equal(10, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf), "10 price overlap")
	t.Log(overlap)
	var j int
	l.buyQueue.Ascend(func(i bt.Item) bool {
		assert.Equal(i.(*BuyPriceLevel).Price, overlap[j].Price, "overlapped Price equal")
		assert.Equal(2*len(i.(*BuyPriceLevel).orders), len(overlap[j].BuyOrders)+len(overlap[j].SellOrders), fmt.Sprintf("order number equal %d", overlap[j].Price))
		j++
		return true
	})

	l.buyQueue = bt.New(8)
	l.InsertOrder("223451", BUYSIDE, 10000, 9750, 1000)
	l.InsertOrder("223452", BUYSIDE, 10000, 9855, 1000)
	l.InsertOrder("223453", BUYSIDE, 10001, 9860, 1000)
	l.InsertOrder("223454", BUYSIDE, 10002, 10001, 1000)
	l.InsertOrder("223455", BUYSIDE, 10002, 10011, 1000)
	l.InsertOrder("223456", BUYSIDE, 10002, 10021, 1000)
	l.InsertOrder("223457", BUYSIDE, 10003, 10021, 1000)
	l.InsertOrder("223458", BUYSIDE, 10003, 10024, 1000)
	l.InsertOrder("223459", BUYSIDE, 10004, 10025, 1000)
	l.InsertOrder("223460", BUYSIDE, 10005, 10031, 1000)
	l.InsertOrder("223461", BUYSIDE, 10005, 10031, 1000)
	l.InsertOrder("223462", BUYSIDE, 10005, 10034, 1000)
	l.InsertOrder("223463", BUYSIDE, 10005, 10035, 1000)
	assert.Equal(17, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf), "10 price overlap")
	t.Log(overlap)
	type PriceOrd struct {
		price int64
		ordNo int
	}
	result := []PriceOrd{{10035, 1}, {10034, 1}, {10033, 1}, {10032, 1}, {10031, 2},
		{10030, 2}, {10025, 1}, {10024, 1}, {10022, 1}, {10021, 3}, {10020, 2},
		{10011, 1}, {10010, 1}, {10001, 1}, {10000, 2}, {9955, 1}, {9950, 1}}

	for j, o := range overlap {
		assert.Equal(o.Price, result[j].price, "overlapped Price equal")
		assert.Equal(len(o.BuyOrders)+len(o.SellOrders), result[j].ordNo, "order number equal")

	}
	l.buyQueue = bt.New(8)
	l.InsertOrder("223451", BUYSIDE, 10000, 9950, 1000)
	l.InsertOrder("223452", BUYSIDE, 10000, 9955, 1000)
	l.InsertOrder("223453", BUYSIDE, 10001, 10000, 1000)
	l.InsertOrder("223454", BUYSIDE, 10002, 10000, 1000)
	l.InsertOrder("223455", BUYSIDE, 10002, 10010, 1000)
	l.InsertOrder("223456", BUYSIDE, 10002, 10020, 1000)
	l.InsertOrder("223457", BUYSIDE, 10003, 10020, 1000)
	l.InsertOrder("223458", BUYSIDE, 10003, 10021, 1000)
	l.InsertOrder("223459", BUYSIDE, 10004, 10022, 1000)
	l.InsertOrder("223460", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223461", BUYSIDE, 10005, 10030, 1000)
	l.InsertOrder("223462", BUYSIDE, 10005, 10032, 1000)
	l.InsertOrder("223463", BUYSIDE, 10005, 10033, 1000)
	l.sellQueue = bt.New(8)
	assert.Equal(0, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf))
	l.InsertOrder("123451", SELLSIDE, 10000, 9750, 1000)
	l.InsertOrder("123452", SELLSIDE, 10000, 9855, 1000)
	l.InsertOrder("123453", SELLSIDE, 10001, 9860, 1000)
	l.InsertOrder("123454", SELLSIDE, 10002, 10000, 1000)
	l.InsertOrder("123455", SELLSIDE, 10002, 10010, 1000)
	l.InsertOrder("123456", SELLSIDE, 10002, 10020, 1000)
	l.InsertOrder("123457", SELLSIDE, 10003, 10020, 1000)
	l.InsertOrder("123458", SELLSIDE, 10003, 10021, 1000)
	l.InsertOrder("123459", SELLSIDE, 10004, 10022, 1000)
	l.InsertOrder("123460", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123461", SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123462", SELLSIDE, 10005, 10132, 1000)
	l.InsertOrder("123463", SELLSIDE, 10005, 10133, 1000)
	assert.Equal(13, l.GetOverlappedRange(&overlap, &buyBuf, &sellBuf), "10 price overlap")
	t.Log(overlap)
	j = 0
	l.buyQueue.Ascend(func(i bt.Item) bool {
		assert.Equal(i.(*BuyPriceLevel).Price, overlap[j].Price, "overlapped Price equal")
		j++
		return true
	})
	assert.Equal(int64(9860), overlap[j].Price)
	assert.Equal(int64(9855), overlap[j+1].Price)
	assert.Equal(int64(9750), overlap[j+2].Price)
}
