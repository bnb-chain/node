package matcheng

import "sort"

/* UnrolledLinkedList (ULList) is implemented here to handle the specific exchange order queue requirement:
1. only handful orders on one of the edges would be touched frequently
 - in the sequence way iteration for match
 - more frequent insert and delete
2. interation across ~hundreds of orders on one of the edges would be frequent
 - publish market data
3. interfation across all orders would be less frequent
 - expire orders
*/

type bucket struct {
	next     *bucket
	elements []PriceLevel
}

func (b *bucket) head() *PriceLevel {
	if len(b.elements) > 0 {
		return &b.elements[0]
	}
	return nil
}

func (b *bucket) size() int {
	return len(b.elements)
}

func (b *bucket) get(p float64, compare Comparator) *PriceLevel {
	i := sort.Search(len(b.elements), func(i int) bool { return compare(b.elements[i].Price, p) >= 0 })
	if i < len(b.elements) && compare(b.elements[i].Price, p) == 0 {
		return &b.elements[i]
	} else {
		return nil
	}
}

func (b *bucket) getRange(p1 float64, p2 float64, compare Comparator, buffer *[]PriceLevel) int {
	// return -1 means the price is out of range
	if len(b.elements) == 0 { // should never reach here
		return -1
	}
	if compare(b.elements[0].Price, p2) < 0 {
		return -1
	}
	if compare(b.elements[len(b.elements)-1].Price, p1) > 0 {
		return 0
	}
	var i int
	for _, p := range b.elements {
		if compare(p2, p.Price) > 0 {
			return -1
		}
		if compare(p1, p.Price) >= 0 {
			*buffer = append(*buffer, p)
			i++
		}
	}
	return i
}

func (b *bucket) insert(p *PriceLevel, compare Comparator) int {
	i := sort.Search(len(b.elements), func(i int) bool { return compare(b.elements[i].Price, p.Price) >= 0 })
	if i == len(b.elements) { // not found
		b.elements = append(b.elements, *p)
		return len(b.elements)
	}
	if compare(b.elements[i].Price, p.Price) == 0 {
		return 0 // duplicated
	}
	b.elements = b.elements[:len(b.elements)+1] //enlarge by 1
	copy(b.elements[i+1:], b.elements[i:])      //shift by 1
	b.elements[i] = *p
	return len(b.elements)
}

func (b *bucket) delete(p float64, compare Comparator) *PriceLevel {
	i := sort.Search(len(b.elements), func(i int) bool { return compare(b.elements[i].Price, p) >= 0 })
	if i == len(b.elements) { // not found
		return nil
	}
	if compare(b.elements[i].Price, p) == 0 {
		pl := &b.elements[i]
		b.elements = append(b.elements[:i], b.elements[i+1:]...)
		return pl
	}
	return nil
}

func (b *bucket) clear() {
	//just reduce the len(), the objects would not be garbage-collected
	b.elements = b.elements[:0]
}

// if p1 < p2, return -1, if p1 == p2 return 0, if p1 > p2, return 1.
type Comparator func(p1 float64, p2 float64) int

type ULList struct {
	begin      *bucket
	dend       *bucket // current data end
	cend       *bucket // real capacity end
	capacity   int
	bucketSize int
	compare    Comparator
	allBuckets []bucket
}

func NewULList(capacity int, bucketSize int, comp Comparator) *ULList {
	if capacity < int(bucketSize) {
		capacity = int(bucketSize)
	}
	bucketNumber := capacity/bucketSize + 1
	realCapacity := bucketNumber * bucketSize
	// pre-allocate everything to make memory adjacency
	allBuckets := make([]bucket, bucketNumber)
	allPriceLevels := make([]PriceLevel, realCapacity)
	var preBucket *bucket = nil
	for i, j := int(0), int(0); i < bucketNumber; i++ {
		//TODO: even allocation may not be the most optimised, should try exponential as well
		allBuckets[i].elements = allPriceLevels[j : j+bucketSize]
		allBuckets[i].elements = allBuckets[i].elements[:0]
		j += bucketSize
		if preBucket != nil {
			preBucket.next = &allBuckets[i]
			preBucket = preBucket.next
		}
	}
	//assert preBucket!=nil
	preBucket.next = nil

	return &ULList{
		&allBuckets[0], //at the very beginnig, only one bucket is used
		&allBuckets[1], //assert bucketNumber > 1
		preBucket,
		capacity,
		bucketSize,
		comp,
		allBuckets}
}

func (ull *ULList) ensureCapacity() {
	if ull.dend == ull.cend { // no empty bucket is available, re-allocate
		oldBucketNumber := ull.capacity/ull.bucketSize + 1
		ull.capacity *= 2
		bucketNumber := ull.capacity/ull.bucketSize + 1
		deltaBucketNumber := bucketNumber - oldBucketNumber
		oldBuckets := ull.allBuckets
		ull.allBuckets = make([]bucket, bucketNumber)
		copy(ull.allBuckets, oldBuckets)
		newPriceLevels := make([]PriceLevel, deltaBucketNumber*ull.bucketSize)
		var preBucket *bucket = nil
		//no need to copy allPriceLevels, since no benefits
		for i, j := oldBucketNumber, int(0); i < bucketNumber; i++ {
			ull.allBuckets[i].elements = newPriceLevels[j : j+ull.bucketSize]
			// clear length
			ull.allBuckets[i].elements = ull.allBuckets[i].elements[:0]
			j += ull.bucketSize
			if preBucket != nil {
				preBucket.next = &ull.allBuckets[i]
				preBucket = preBucket.next
			}
		}
		preBucket.next = nil
		ull.dend.next = &ull.allBuckets[oldBucketNumber]
		ull.cend = preBucket
	}
}

func (ull *ULList) splitBucket(origin *bucket) *bucket {
	ull.ensureCapacity()
	//assert(ull.dend!=ull.cend), i.e. there is still avaiable free bucket
	oldNext := origin.next
	origin.next = ull.dend
	ull.dend = ull.dend.next
	origin.next.next = oldNext
	oldElements := origin.elements
	newElements := origin.next.elements
	origin.next.elements = append(newElements, oldElements[len(oldElements)/2:]...)
	origin.elements = oldElements[:len(oldElements)/2]
	return origin.next
}

func (ull *ULList) Clear() {
	for i := ull.begin; i != ull.dend; i = i.next {
		i.clear()
	}
	ull.dend = ull.begin.next // only leave with one bucket
}

func (ull *ULList) getBucket(p float64) *bucket {
	var last *bucket = nil
	for b := ull.begin; b != ull.dend; b = b.next {
		h := b.head()
		if h != nil && ull.compare(h.Price, p) == -1 {
			break
		}
		last = b
	}
	return last
}

func (ull *ULList) SetPriceLevel(p *PriceLevel) bool {
	last := ull.getBucket(p.Price)
	if last == nil {
		last = ull.begin
	}
	if last.size() >= ull.bucketSize {
		//bucket is full, split
		//TODO: do we have to wait until it is 100% full?
		ull.splitBucket(last)
	}
	return last.insert(p, ull.compare) > 0
}

func (ull *ULList) DeletePriceLevel(price float64) bool {
	var last, lastOfLast *bucket
	for b := ull.begin; b != ull.dend; b = b.next {
		h := b.head()
		if h != nil && ull.compare(h.Price, price) == -1 {
			break
		}
		lastOfLast = last
		last = b
	}
	if last == nil {
		//not found
		return false
	}
	if last.delete(price, ull.compare) != nil {
		if last.size() == 0 {
			// bucket is empty, remove from list
			oldNext := last.next
			if lastOfLast == nil { //i.e. last == ull.begin
				ull.begin = oldNext
			} else {
				lastOfLast.next = oldNext
			}
			// change dend instead of cend, so that it may be swapped in soon
			oldDataEnd := ull.dend
			ull.dend = last
			last.next = oldDataEnd
		}
		return true
	}
	return false
}

func (ull *ULList) GetTop() *PriceLevel {
	return ull.begin.head()
}

func (ull *ULList) GetPriceRange(p1 float64, p2 float64, buffer *[]PriceLevel) []PriceLevel {
	ret := (*buffer)[:0]
	if ull.compare(p1, p2) < 0 || len(ull.begin.elements) <= 0 {
		return ret // empty slice
	}
	if ull.compare(ull.begin.head().Price, p2) < 0 {
		return ret
	}

	for i := ull.begin; i != ull.dend; i = i.next {
		if i.getRange(p1, p2, ull.compare, buffer) < 0 {
			return *buffer
		}
	}
	return *buffer
}

func (ull *ULList) GetPriceLevel(p float64) *PriceLevel {
	for i := ull.begin; i != ull.dend; i = i.next {
		h := i.head()
		if h != nil {
			c := ull.compare(h.Price, p)
			switch c {
			case 0: //head has the price
				return h
			case 1: //head is less
				if i == ull.dend { // last bucket
					return i.get(p, ull.compare)
				}
				h = i.next.head()
				// next bucket is more
				if h != nil && ull.compare(h.Price, p) == -1 {
					return i.get(p, ull.compare)
				}
				//continue to move to the next bucket
			case -1: // no way to reach here
				return nil
			}
		} else {
			return nil
		}
	}
	return nil
}
