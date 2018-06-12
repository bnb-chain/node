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
				if ull.begin != &ull.allBuckets[0] || ull.dend != &ull.allBuckets[1] {
					t.Error("data end / begin is not correct in NewULList")
				}
				if ull.begin.size() != 0 {
					t.Error("NewULList initial size is not zero")
				}
				var i int
				for k := ull.begin; k.next != nil; k = k.next {
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
		capacity   int
		bucketSize int
		compare    Comparator
		allBuckets []bucket
	}
	type args struct {
		p float64
	}

	buys := []PriceLevel{{Price: 100.5},
		{Price: 100.2},
		{Price: 100.1},
		{Price: 99.5},
		{Price: 99.4}}
	sells := []PriceLevel{{Price: 100.0},
		{Price: 101.2},
		{Price: 102.1},
		{Price: 102.5},
		{Price: 103.4}}

	makeFields := func(levels []PriceLevel, n int) *fields {
		allBuckets := make([]bucket, 4)
		begin := &allBuckets[0]
		for i := 0; i < 3; i++ {
			allBuckets[i].elements = allBuckets[i].elements[:0]
			allBuckets[i].next = &allBuckets[i+1]
		}
		allBuckets[3].next = nil
		dend := &allBuckets[3]
		switch n {
		case 1:
			allBuckets[0].elements = levels[:1]
			allBuckets[1].elements = levels[1:3]
			allBuckets[2].elements = levels[3:]
			return &fields{begin, dend, 8, 2, compareBuy, allBuckets}
		case 2:
			allBuckets[0].elements = levels[:2]
			allBuckets[1].elements = levels[2:4]
			allBuckets[2].elements = levels[4:]
			return &fields{begin, dend, 8, 2, compareSell, allBuckets}
		}
		return &fields{begin, dend, 8, 2, compareBuy, allBuckets}
	}
	field1 := *makeFields(buys, 1)
	field2 := *makeFields(sells, 2)
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *PriceLevel
	}{
		{"NotExist1", field1, args{99.0}, nil},
		{"NotExist2", field2, args{99.0}, nil},
		{"NotExist3", field1, args{105.0}, nil},
		{"NotExist4", field2, args{105.0}, nil},
		{"NotExist5", field1, args{100.11}, nil},
		{"NotExist6", field2, args{100.11}, nil},
		{"Exist1", field1, args{100.5}, &PriceLevel{Price: 100.5}},
		{"Exist2", field2, args{100.0}, &PriceLevel{Price: 100.0}},
		{"Exist3", field1, args{99.4}, &PriceLevel{Price: 99.4}},
		{"Exist4", field2, args{103.4}, &PriceLevel{Price: 103.4}},
		{"Exist5", field1, args{100.2}, &PriceLevel{Price: 100.2}},
		{"Exist6", field2, args{102.5}, &PriceLevel{Price: 102.5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ull := &ULList{
				begin:      tt.fields.begin,
				dend:       tt.fields.dend,
				capacity:   tt.fields.capacity,
				bucketSize: tt.fields.bucketSize,
				compare:    tt.fields.compare,
				allBuckets: tt.fields.allBuckets,
			}
			t.Logf("before GetPriceLevel: %v", ull)
			if got := ull.GetPriceLevel(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Logf("after GetPriceLevel: %v", ull)
				t.Errorf("ULList.GetPriceLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestULList_SetPriceLevel(t *testing.T) {
	type fields struct {
		begin      *bucket
		dend       *bucket
		capacity   int
		bucketSize int
		compare    Comparator
		allBuckets []bucket
	}
	type args struct {
		p *PriceLevel
	}

	makeFields := func(n int) *fields {
		allBuckets := make([]bucket, 4)
		begin := &allBuckets[0]
		for i := 0; i < 3; i++ {
			allBuckets[i].elements = allBuckets[i].elements[:0]
			allBuckets[i].next = &allBuckets[i+1]
		}
		allBuckets[3].next = nil
		dend := &allBuckets[3]
		switch n {
		case 1:
			levels := []PriceLevel{{Price: 100.5},
				{Price: 100.2},
				{Price: 100.1},
				{Price: 99.5},
				{Price: 99.4}}
			allBuckets[0].elements = levels[:1]
			allBuckets[1].elements = levels[1:3]
			allBuckets[2].elements = levels[3:]
			return &fields{begin, dend, 8, 2, compareBuy, allBuckets}
		case 2:
			levels := []PriceLevel{{Price: 100.0},
				{Price: 101.2},
				{Price: 102.1},
				{Price: 102.5},
				{Price: 103.4}}
			allBuckets[0].elements = levels[:2]
			allBuckets[1].elements = levels[2:4]
			allBuckets[2].elements = levels[4:]
			return &fields{begin, dend, 8, 2, compareSell, allBuckets}
		}
		return &fields{begin, dend, 8, 2, compareBuy, allBuckets}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{"NotExist1", *makeFields(1), args{&PriceLevel{Price: 99.0}}, true},
		{"NotExist2", *makeFields(2), args{&PriceLevel{Price: 99.0}}, true},
		{"NotExist3", *makeFields(1), args{&PriceLevel{Price: 105.0}}, true},
		{"NotExist4", *makeFields(2), args{&PriceLevel{Price: 105.0}}, true},
		{"NotExist5", *makeFields(1), args{&PriceLevel{Price: 100.11}}, true},
		{"NotExist6", *makeFields(2), args{&PriceLevel{Price: 100.11}}, true},
		// one side
		{"Exist1", *makeFields(1), args{&PriceLevel{Price: 100.51}}, true},
		{"Exist2", *makeFields(2), args{&PriceLevel{Price: 100.110}}, true},
		{"Exist3", *makeFields(1), args{&PriceLevel{Price: 99.3}}, true},
		{"Exist4", *makeFields(2), args{&PriceLevel{Price: 103.5}}, true},
		// in the middle
		{"Exist5", *makeFields(1), args{&PriceLevel{Price: 100.21}}, true},
		{"Exist6", *makeFields(2), args{&PriceLevel{Price: 102.35}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ull := &ULList{
				begin:      tt.fields.begin,
				dend:       tt.fields.dend,
				capacity:   tt.fields.capacity,
				bucketSize: tt.fields.bucketSize,
				compare:    tt.fields.compare,
				allBuckets: tt.fields.allBuckets,
			}
			if got := ull.AddPriceLevel(tt.args.p); got != tt.want {
				t.Errorf("ULList.AddPriceLevel() = %v, want %v", got, tt.want)
			}
			t.Log(ull.allBuckets)
		})
	}
}

func TestULList_ensureCapacity(t *testing.T) {
	type fields struct {
		begin      *bucket
		dend       *bucket
		capacity   int
		bucketSize int
		compare    Comparator
		allBuckets []bucket
	}
	makeFields := func() *fields {
		allBuckets := make([]bucket, 4)
		begin := &allBuckets[0]
		for i := 0; i < 3; i++ {
			allBuckets[i].next = &allBuckets[i+1]
		}
		dend := &allBuckets[3]
		dend.next = nil
		bucketSize := 2
		capacity := 6
		return &fields{begin, dend, capacity, bucketSize, compareBuy, allBuckets}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"Full", *makeFields()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ull := &ULList{
				begin:      tt.fields.begin,
				dend:       tt.fields.dend,
				capacity:   tt.fields.capacity,
				bucketSize: tt.fields.bucketSize,
				compare:    tt.fields.compare,
				allBuckets: tt.fields.allBuckets,
			}
			ull.ensureCapacity()
			oldDend := &ull.allBuckets[3]
			i, j := 0, 0
			for k := ull.begin; k != ull.dend; k = k.next {
				i++
			}
			for k := ull.begin; k.next != nil; k = k.next {
				j++
			}
			if ull.capacity != 12 || ull.dend.next == nil ||
				ull.dend == oldDend || len(ull.allBuckets) != 7 || j != 6 || i != 3 {
				t.Errorf("Re-allocate failed: capacity=%d, allBuckets=%d, dend(%p)/allBuckets[3](%p), i=%d, j=%d",
					ull.capacity, len(ull.allBuckets), ull.dend, oldDend, i, j)
			}
		})
	}
}
