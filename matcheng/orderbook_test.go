package matcheng

import (
	"reflect"
	"testing"

	bt "github.com/google/btree"
)

func Test_toPriceLevel(t *testing.T) {
	type args struct {
		pi   PriceLevelInterface
		side int
	}
	tests := []struct {
		name string
		args args
		want *PriceLevel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toPriceLevel(tt.args.pi, tt.args.side); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toPriceLevel() = %v, want %v", got, tt.want)
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
			ob := &OrderBookOnULList{
				buyQueue:  tt.fields.buyQueue,
				sellQueue: tt.fields.sellQueue,
			}
			got, err := ob.InsertOrder(tt.args.id, tt.args.side, tt.args.time, tt.args.price, tt.args.qty)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderBookOnULList.InsertOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrderBookOnULList.InsertOrder() = %v, want %v", got, tt.want)
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
