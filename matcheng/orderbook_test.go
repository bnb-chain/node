package matcheng

import (
	"reflect"
	"strings"
	"testing"

	bt "github.com/google/btree"
)

func Test_compareBuy(t *testing.T) {
	type args struct {
		p1 float64
		p2 float64
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"P1Bigger1", args{0.005111, 0.005}, 1},
		{"P1Bigger2", args{1.005, 0.005}, 1},
		{"P1Bigger3", args{0.00000111, 0.0000011}, 1},
		{"P1Bigger4", args{0.00000001, 0.0}, 1},
		{"Equal1", args{0.000000001, 0.0}, 0},
		{"Equal2", args{-0.000000001, 0.0}, 0},
		{"Equal3", args{-1.000000001, -1.0}, 0},
		{"Equal4", args{5.581234, 5.581234}, 0},
		{"Equal5", args{5.581234567, 5.581234568}, 0},
		{"P2Bigger1", args{0.005, 0.005111}, -1},
		{"P2Bigger2", args{0.005, 1.005}, -1},
		{"P2Bigger3", args{0.0000011, 0.00000111}, -1},
		{"P2Bigger4", args{0.0, 0.00000001}, -1},
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
		Price  float64
		orders []OrderPart
	}
	type args struct {
		id   string
		time uint64
		qty  float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{"AddedOrder", fields{100.0, make([]OrderPart, 0, 1)}, args{"12345", 2354, 1000.5}, 1, false},
		{"Duplicated", fields{100.0, []OrderPart{{"12345", 0, 1555}}}, args{"12345", 2354, 1000.5}, 0, true},
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
		Price  float64
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
		{"NotExist1", fields{100.0, []OrderPart{{"12345", 0, 1555}}}, args{"12346"}, OrderPart{}, 0, true},
		{"NotExist2", fields{100.0, []OrderPart{}}, args{"12346"}, OrderPart{}, 0, true},
		{"Delete1", fields{100.0, []OrderPart{{"12345", 0, 1555}, {"12346", 0, 1556}, {"12347", 0, 1557}}}, args{"12345"}, OrderPart{"12345", 0, 1555}, 1, false},
		{"Delete2", fields{100.0, []OrderPart{{"12345", 0, 1555}, {"12346", 0, 1556}, {"12347", 0, 1557}}}, args{"12347"}, OrderPart{"12347", 0, 1557}, 1, false},
		{"Delete3", fields{100.0, []OrderPart{{"12345", 0, 1555}, {"12346", 0, 1556}, {"12347", 0, 1557}}}, args{"12346"}, OrderPart{"12346", 0, 1556}, 1, false},
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
		{"OneSide1", args{[]PriceLevel{{Price: 120.0}, {Price: 101.0}, {Price: 100.0}}, nil, &overLapped}},
		{"OneSide2", args{nil, []PriceLevel{{Price: 100.0}, {Price: 101.0}, {Price: 120.0}}, &overLapped}},
		{"OneOnEachSide", args{[]PriceLevel{{Price: 101.0}}, []PriceLevel{{Price: 100.0}}, &overLapped}},
		{"NormalMerge", args{[]PriceLevel{{Price: 104.1, orders: make([]OrderPart, 3)}, {Price: 103.0}}, []PriceLevel{{Price: 102.0}, {Price: 103.2, orders: make([]OrderPart, 4)}}, &overLapped}},
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
				if len(overLapped) != 3 || overLapped[0].Price != 120.0 {
					t.Error("mergeLevel failed to merage one side")
				}
			}
			if strings.Contains(tt.name, "OneOnEach") {
				if len(overLapped) != 2 || overLapped[0].Price != 101.0 || overLapped[1].Price != 100.0 {
					t.Errorf("mergeLevel failed to merage one on each side %v, %v, %v",
						len(overLapped), overLapped[0].Price, overLapped[1].Price)
				}
			}
			if strings.Contains(tt.name, "Normal") {
				if len(overLapped) != 4 || overLapped[0].Price != 104.1 || overLapped[1].Price != 103.2 {
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
		time  uint64
		price float64
		qty   float64
	}

	samePrice := func() *OrderBookOnULList {
		l := NewOrderBookOnULList(4)
		l.InsertOrder("123455", BUYSIDE, 10000, 100.0, 1000)
		l.InsertOrder("123457", BUYSIDE, 10001, 100.0, 1000)
		l.InsertOrder("123458", BUYSIDE, 10002, 100.0, 1000)
		return l
	}()
	newPrice := func() *OrderBookOnULList {
		l := NewOrderBookOnULList(4)
		l.InsertOrder("123459", BUYSIDE, 10002, 100.5, 1000)
		l.InsertOrder("123459", BUYSIDE, 10002, 99.5, 1000)
		l.InsertOrder("123455", BUYSIDE, 10000, 100.0, 1000)
		l.InsertOrder("123458", BUYSIDE, 10002, 100.0, 1000)
		return l
	}()
	/* 	newPrice2 := func() *OrderBookOnULList {
		l := NewOrderBookOnULList(4)
		l.InsertOrder("123459", BUYSIDE, 10002, 100.5, 1000)
		l.InsertOrder("123459", BUYSIDE, 10002, 99.5, 1000)
		l.InsertOrder("123455", BUYSIDE, 10000, 100.0, 1000)
		l.InsertOrder("123457", BUYSIDE, 10001, 100.7, 1000)
		l.InsertOrder("123458", BUYSIDE, 10002, 100.0, 1000)
		l.InsertOrder("123458", BUYSIDE, 10002, 100.8, 1000)
		return l
	}() */
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *PriceLevel
		wantErr bool
	}{
		{"Sanity", fields{NewULList(4096, 16, compareBuy), NewULList(4096, 16, compareSell)},
			args{"123456", BUYSIDE, 10000, 100.0, 1000.0}, &PriceLevel{100.0, []OrderPart{{"123456", 10000, 1000.0}}}, false},
		{"SamePrice", fields{samePrice.buyQueue, samePrice.sellQueue},
			args{"123456", BUYSIDE, 10000, 100.0, 1000.0}, &PriceLevel{100.0, []OrderPart{{"123455", 10000, 1000.0}, {"123457", 10001, 1000.0}, {"123458", 10002, 1000.0}, {"123456", 10000, 1000.0}}}, false},
		{"NewPrice1", fields{newPrice.buyQueue, newPrice.sellQueue},
			args{"123456", BUYSIDE, 10000, 101.0, 1000.0}, &PriceLevel{101.0, []OrderPart{{"123456", 10000, 1000.0}}}, false},
		/* {"NewPrice2", fields{newPrice.buyQueue, newPrice.sellQueue},
			args{"123456", BUYSIDE, 10000, 99.0, 1000.0}, &PriceLevel{99.0, []OrderPart{{"123456", 10000, 1000.0}}}, false},
				{"NewPrice3", fields{newPrice2.buyQueue, newPrice2.sellQueue},
		   			args{"123456", BUYSIDE, 10000, 101.0, 1000.0}, &PriceLevel{101.0, []OrderPart{{"123456", 10000, 1000.0}}}, false},
		   		{"NewPrice4", fields{newPrice2.buyQueue, newPrice2.sellQueue},
		   			args{"123456", BUYSIDE, 10000, 99.0, 1000.0}, &PriceLevel{99.0, []OrderPart{{"123456", 10000, 1000.0}}}, false}, */
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBookOnULList{
				buyQueue:  tt.fields.buyQueue,
				sellQueue: tt.fields.sellQueue,
			}
			got, err := ob.InsertOrder(tt.args.id, tt.args.side, tt.args.time, tt.args.price, tt.args.qty)
			t.Logf("after insert:%v", ob)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderBookOnULList.InsertOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrderBookOnULList.InsertOrder() = %v, want %v", got, tt.want)
			}
			switch tt.name {

			case "SamePrice":

				if len(ob.sellQueue.begin.elements) != 0 || len(ob.buyQueue.begin.elements) != 1 ||
					len(ob.buyQueue.begin.elements[0].orders) != 4 ||
					ob.buyQueue.begin.elements[0].orders[0].id != "123455" ||
					ob.buyQueue.begin.elements[0].orders[3].id != "123456" {
					t.Error("SamePrice doesn't work")
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
		time  uint64
		price float64
		qty   float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *PriceLevel
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBookOnBTree{
				buyQueue:  tt.fields.buyQueue,
				sellQueue: tt.fields.sellQueue,
			}
			got, err := ob.InsertOrder(tt.args.id, tt.args.side, tt.args.time, tt.args.price, tt.args.qty)
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
