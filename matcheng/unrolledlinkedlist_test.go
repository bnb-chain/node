package matcheng

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
	buyBucket := fields{nil, []PriceLevel{{Price: 1010}, {Price: 1000}, {Price: 990}, {Price: 989}, {Price: 988}}}
	sellBucket := fields{nil, []PriceLevel{{Price: 998}, {Price: 999}, {Price: 1990}, {Price: 1991}, {Price: 1992}}}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{"NilBucket", nilBucket, args{&PriceLevel{Price: 1011}, compareBuy}, 1},
		{"NilBucket2", nilBucket, args{&PriceLevel{Price: 1011}, compareSell}, 1},
		{"EmptyBucket", emptyBucket, args{&PriceLevel{Price: 1011}, compareBuy}, 1},
		{"EmptyBucket2", emptyBucket, args{&PriceLevel{Price: 1011}, compareSell}, 1},
		{"BuyBucket1", buyBucket, args{&PriceLevel{Price: 1011}, compareBuy}, 6},
		{"BuyBucket2", buyBucket, args{&PriceLevel{Price: 1001}, compareBuy}, 6},
		{"BuyBucket3", buyBucket, args{&PriceLevel{Price: 991}, compareBuy}, 6},
		{"BuyBucket4", buyBucket, args{&PriceLevel{Price: 971}, compareBuy}, 6},
		{"BuyBucket5", buyBucket, args{&PriceLevel{Price: 988}, compareBuy}, 0},
		{"SellBucket1", sellBucket, args{&PriceLevel{Price: 991}, compareSell}, 6},
		{"SellBucket2", sellBucket, args{&PriceLevel{Price: 100}, compareSell}, 6},
		{"SellBucket3", sellBucket, args{&PriceLevel{Price: 1992}, compareSell}, 0},
		{"SellBucket4", sellBucket, args{&PriceLevel{Price: 11991}, compareSell}, 6},
		{"SellBucket5", sellBucket, args{&PriceLevel{Price: 998}, compareSell}, 0},
		{"SellBucket6", sellBucket, args{&PriceLevel{Price: 999}, compareSell}, 0},
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
			switch tt.name {
			case "NilBucket", "NilBucket2", "EmptyBucket", "EmptyBucket2":
				if b.elements[0].Price != 1011 {
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
		p int64
	}

	buys := []PriceLevel{{Price: 1005},
		{Price: 1002},
		{Price: 1001},
		{Price: 995},
		{Price: 994}}
	sells := []PriceLevel{{Price: 1000},
		{Price: 1012},
		{Price: 1021},
		{Price: 1025},
		{Price: 1034}}

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
		{"NotExist1", field1, args{990}, nil},
		{"NotExist2", field2, args{990}, nil},
		{"NotExist3", field1, args{1050}, nil},
		{"NotExist4", field2, args{1050}, nil},
		{"NotExist5", field1, args{1000}, nil},
		{"NotExist6", field2, args{1001}, nil},
		{"Exist1", field1, args{1005}, &PriceLevel{Price: 1005}},
		{"Exist2", field2, args{1000}, &PriceLevel{Price: 1000}},
		{"Exist3", field1, args{994}, &PriceLevel{Price: 994}},
		{"Exist4", field2, args{1034}, &PriceLevel{Price: 1034}},
		{"Exist5", field1, args{1002}, &PriceLevel{Price: 1002}},
		{"Exist6", field2, args{1025}, &PriceLevel{Price: 1025}},
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
			levels := []PriceLevel{{Price: 1005},
				{Price: 1002},
				{Price: 1001},
				{Price: 995},
				{Price: 994}}
			allBuckets[0].elements = levels[:1]
			allBuckets[1].elements = levels[1:3]
			allBuckets[2].elements = levels[3:]
			return &fields{begin, dend, 8, 2, compareBuy, allBuckets}
		case 2:
			levels := []PriceLevel{{Price: 1000},
				{Price: 1012},
				{Price: 1021},
				{Price: 1025},
				{Price: 1034}}
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
		{"NotExist1", *makeFields(1), args{&PriceLevel{Price: 990}}, true},
		{"NotExist2", *makeFields(2), args{&PriceLevel{Price: 990}}, true},
		{"NotExist3", *makeFields(1), args{&PriceLevel{Price: 1050}}, true},
		{"NotExist4", *makeFields(2), args{&PriceLevel{Price: 1050}}, true},
		{"NotExist5", *makeFields(1), args{&PriceLevel{Price: 1003}}, true},
		{"NotExist6", *makeFields(2), args{&PriceLevel{Price: 1001}}, true},
		// one side
		{"Exist1", *makeFields(1), args{&PriceLevel{Price: 1005}}, false},
		{"Exist2", *makeFields(1), args{&PriceLevel{Price: 999}}, true},
		{"Exist3", *makeFields(1), args{&PriceLevel{Price: 993}}, true},
		{"Exist4", *makeFields(2), args{&PriceLevel{Price: 1034}}, false},
		// in the middle
		{"Exist5", *makeFields(1), args{&PriceLevel{Price: 1002}}, false},
		{"Exist6", *makeFields(2), args{&PriceLevel{Price: 1023}}, true},
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

func TestULList_DeletePriceLevel(t *testing.T) {
	assert := assert.New(t)
	l := NewULList(5, 2, compareBuy)
	l.AddPriceLevel(&PriceLevel{Price: 1006})
	l.AddPriceLevel(&PriceLevel{Price: 1002})
	l.AddPriceLevel(&PriceLevel{Price: 1003})
	l.AddPriceLevel(&PriceLevel{Price: 1001})
	assert.Equal("Bucket 0{1006->[]},Bucket 1{1003->[]1002->[]},Bucket 2{1001->[]},", l.String(), "AddPriceLevel sequence is wrong")
	l.DeletePriceLevel(1003)
	assert.Equal("Bucket 0{1006->[]},Bucket 1{1002->[]},Bucket 2{1001->[]},", l.String(), "Delete mid price")
	l.DeletePriceLevel(1002)
	assert.Equal("Bucket 0{1006->[]},Bucket 1{1001->[]},", l.String(), "Delete mid price")
	l.DeletePriceLevel(1006)
	assert.Equal("Bucket 0{1001->[]},", l.String(), "Delete 1st bucket")
	l.AddPriceLevel(&PriceLevel{Price: 1006})
	l.AddPriceLevel(&PriceLevel{Price: 1002})
	assert.Equal("Bucket 0{1006->[]},Bucket 1{1002->[]1001->[]},", l.String(), "split bucket for new price")
	l.DeletePriceLevel(1001)
	assert.Equal("Bucket 0{1006->[]},Bucket 1{1002->[]},", l.String(), "Delete price from last bucket")
	l.DeletePriceLevel(1002)
	assert.Equal("Bucket 0{1006->[]},", l.String(), "Delete last bucket")
	l.DeletePriceLevel(1006)
	assert.Equal("", l.String(), "Delete last price")
	assert.False(l.DeletePriceLevel(1006), "delete empty")
}

func Test_bucket_getRange(t *testing.T) {
	assert := assert.New(t)
	b1 := bucket{nil, []PriceLevel{
		PriceLevel{Price: 1060},
		PriceLevel{Price: 1050},
		PriceLevel{Price: 1040},
		PriceLevel{Price: 1030},
		PriceLevel{Price: 1020},
		PriceLevel{Price: 1010},
		PriceLevel{Price: 1000},
	}}
	buyBuf := make([]PriceLevel, 16)
	assert.Equal(0, b1.getRange(910, 800, compareBuy, &buyBuf), "no overlap")
	assert.Equal(-1, b1.getRange(1080, 1070, compareBuy, &buyBuf), "no overlap")
	assert.Equal(-1, b1.getRange(1000, 1070, compareBuy, &buyBuf), "no overlap")
	assert.Equal(1, b1.getRange(1000, 1000, compareBuy, &buyBuf), "1 overlap")
	assert.Equal(int64(1000), buyBuf[len(buyBuf)-1].Price, "100 equal")
	assert.Equal(1, b1.getRange(1060, 1060, compareBuy, &buyBuf), "106 overlap")
	assert.Equal(int64(1060), buyBuf[len(buyBuf)-1].Price, "106 equal")
	assert.Equal(1, b1.getRange(1030, 1030, compareBuy, &buyBuf), "1 overlap")
	assert.Equal(int64(1030), buyBuf[len(buyBuf)-1].Price, "103 equal")
	assert.Equal(2, b1.getRange(1060, 1050, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(1, b1.getRange(1060, 1055, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(2, b1.getRange(1080, 1050, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(2, b1.getRange(1080, 1045, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(2, b1.getRange(1055, 1035, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(3, b1.getRange(1050, 1030, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(3, b1.getRange(1050, 1026, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(3, b1.getRange(1020, 1000, compareBuy, &buyBuf), "2 overlap")
	assert.Equal(3, b1.getRange(1025, 990, compareBuy, &buyBuf), "2 overlap")
	b2 := bucket{nil, []PriceLevel{
		PriceLevel{Price: 1000},
		PriceLevel{Price: 1010},
		PriceLevel{Price: 1020},
		PriceLevel{Price: 1030},
		PriceLevel{Price: 1040},
		PriceLevel{Price: 1050},
		PriceLevel{Price: 1060},
	}}
	assert.Equal(-1, b2.getRange(810, 900, compareSell, &buyBuf), "no overlap")
	assert.Equal(0, b2.getRange(1070, 1080, compareSell, &buyBuf), "no overlap")
	assert.Equal(-1, b2.getRange(1100, 1070, compareSell, &buyBuf), "no overlap")
	assert.Equal(1, b2.getRange(1000, 1000, compareSell, &buyBuf), "1 overlap")
	assert.Equal(1, b2.getRange(1060, 1060, compareSell, &buyBuf), "1 overlap")
	assert.Equal(1, b2.getRange(1030, 1030, compareSell, &buyBuf), "1 overlap")
	assert.Equal(2, b2.getRange(1050, 1060, compareSell, &buyBuf), "2 overlap")
	assert.Equal(1, b2.getRange(1056, 1060, compareSell, &buyBuf), "2 overlap")
	assert.Equal(2, b2.getRange(1050, 1080, compareSell, &buyBuf), "2 overlap")
	assert.Equal(2, b2.getRange(1045, 1080, compareSell, &buyBuf), "2 overlap")
	assert.Equal(2, b2.getRange(1035, 1055, compareSell, &buyBuf), "2 overlap")
	assert.Equal(3, b2.getRange(1030, 1050, compareSell, &buyBuf), "2 overlap")
	assert.Equal(3, b2.getRange(1026, 1050, compareSell, &buyBuf), "2 overlap")
	assert.Equal(3, b2.getRange(1000, 1020, compareSell, &buyBuf), "2 overlap")
	assert.Equal(3, b2.getRange(990, 1025, compareSell, &buyBuf), "2 overlap")

}
