package matcheng

import (
	"reflect"
	"testing"
)

func Test_bucket_head(t *testing.T) {
	type fields struct {
		next     *bucket
		elements []PriceLevel
	}
	es := make([]PriceLevel, 4)
	tests := []struct {
		name   string
		fields fields
		want   *PriceLevel
	}{
		{"EmptyBucket", fields{nil, make([]PriceLevel, 0, 16)}, nil},
		{"BucketHead", fields{nil, es}, &es[0]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bucket{
				next:     tt.fields.next,
				elements: tt.fields.elements,
			}
			if got := b.head(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bucket.head() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewULList(t *testing.T) {
	type args struct {
		capacity   int
		bucketSize int
		comp       Comparator
	}
	tests := []struct {
		name string
		args args
	}{
		{"Sanity", args{4096, 16, compareBuy}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ull := NewULList(tt.args.capacity, tt.args.bucketSize, tt.args.comp)
			switch tt.name {
			case "Sanity":
				if ull.bucketSize != 16 || ull.capacity != 4096 {
					t.Error("Wrong size / capacity for the NewULList")
				}
				if ull.begin != &ull.allBuckets[0] || ull.dend != &ull.allBuckets[1] ||
					ull.cend != &ull.allBuckets[4096/16] {
					t.Error("data end / begin is not correct in NewULList")
				}
				if ull.begin.size() != 0 {
					t.Error("NewULList initial size is not zero")
				}
				var i int
				for k := ull.begin; k != ull.cend; k = k.next {
					i++
				}
				if i != 4096/16 {
					t.Errorf("NewULList linked bucket number %v is not wanted:%v", i, 4096/16)
				}
			}
		})
	}
}

func Test_bucket_insert(t *testing.T) {
	type fields struct {
		next     *bucket
		elements []PriceLevel
	}
	type args struct {
		p       *PriceLevel
		compare Comparator
	}
	nilBucket := fields{nil, nil}
	emptyBucket := fields{nil, make([]PriceLevel, 0)}
	buyBucket := fields{nil, []PriceLevel{{Price: 101.0}, {Price: 100.0}, {Price: 99.0}, {Price: 98.9}, {Price: 98.8999}}}
	sellBucket := fields{nil, []PriceLevel{{Price: 99.8999}, {Price: 99.9}, {Price: 199.0}, {Price: 199.1}, {Price: 199.11}}}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{"NilBucket", nilBucket, args{&PriceLevel{Price: 101.1}, compareBuy}, 1},
		{"NilBucket2", nilBucket, args{&PriceLevel{Price: 101.1}, compareSell}, 1},
		{"EmptyBucket", emptyBucket, args{&PriceLevel{Price: 101.1}, compareBuy}, 1},
		{"EmptyBucket2", emptyBucket, args{&PriceLevel{Price: 101.1}, compareSell}, 1},
		{"BuyBucket1", buyBucket, args{&PriceLevel{Price: 101.1}, compareBuy}, 6},
		{"BuyBucket2", buyBucket, args{&PriceLevel{Price: 100.1}, compareBuy}, 6},
		{"BuyBucket3", buyBucket, args{&PriceLevel{Price: 99.1}, compareBuy}, 6},
		{"BuyBucket4", buyBucket, args{&PriceLevel{Price: 97.1}, compareBuy}, 6},
		{"BuyBucket5", buyBucket, args{&PriceLevel{Price: 98.8999}, compareBuy}, 0},
		{"SellBucket1", sellBucket, args{&PriceLevel{Price: 99.1}, compareSell}, 6},
		{"SellBucket2", sellBucket, args{&PriceLevel{Price: 99.89999}, compareSell}, 6},
		{"SellBucket3", sellBucket, args{&PriceLevel{Price: 199.2}, compareSell}, 6},
		{"SellBucket4", sellBucket, args{&PriceLevel{Price: 1199.1}, compareSell}, 6},
		{"SellBucket5", sellBucket, args{&PriceLevel{Price: 99.8999}, compareSell}, 0},
		{"SellBucket6", sellBucket, args{&PriceLevel{Price: 99.9}, compareSell}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bucket{
				next:     tt.fields.next,
				elements: tt.fields.elements,
			}
			if got := b.insert(tt.args.p, tt.args.compare); got != tt.want {
				t.Errorf("bucket.insert() = %v, want %v", got, tt.want)
			}
			t.Log(b)
			switch tt.name {
			case "NilBucket", "NilBucket2", "EmptyBucket", "EmptyBucket2":
				if b.elements[0].Price != 101.1 {
					t.Error("bucket.insert failed")
				}
			case "BuyBucket1", "BuyBucket2", "BuyBucket3", "BuyBucket4":
				func() {
					for i := 0; i < len(b.elements)-2; i++ {
						if compareBuy(b.elements[i].Price, b.elements[i+1].Price) <= 0 {
							t.Errorf("bucket.insert not sorted: %v < %v", b.elements[i], b.elements[i+1])
						}
					}
				}()
			}
		})
	}
}

func TestULList_GetPriceLevel(t *testing.T) {
	type fields struct {
		begin      *bucket
		dend       *bucket
		cend       *bucket
		capacity   int
		bucketSize int
		compare    Comparator
		allBuckets []bucket
	}
	type args struct {
		p float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *PriceLevel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ull := &ULList{
				begin:      tt.fields.begin,
				dend:       tt.fields.dend,
				cend:       tt.fields.cend,
				capacity:   tt.fields.capacity,
				bucketSize: tt.fields.bucketSize,
				compare:    tt.fields.compare,
				allBuckets: tt.fields.allBuckets,
			}
			if got := ull.GetPriceLevel(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ULList.GetPriceLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
