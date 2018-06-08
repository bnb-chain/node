package matcheng

import (
	"bytes"
	"fmt"
	"sort"
)

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
	k := len(b.elements)
	i := sort.Search(k, func(i int) bool { return compare(b.elements[i].Price, p) == 0 })
	if i < k {
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
	k := len(b.elements)
	i := sort.Search(k, func(i int) bool { return compare(b.elements[i].Price, p.Price) < 0 })
	if i > 0 && compare(b.elements[i-1].Price, p.Price) == 0 {
		//TODO: overwrite?
		return 0 // duplicated
	}
	if i == k { // not found
		b.elements = append(b.elements, *p)
		return len(b.elements)
	}
	b.elements = append(b.elements, b.elements[k-1]) //enlarge by 1
	copy(b.elements[i+1:], b.elements[i:])           //shift by 1
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
	capacity   int
	bucketSize int
	compare    Comparator
	allBuckets []bucket
}

func NewULList(capacity int, bucketSize int, comp Comparator) *ULList {
	if bucketSize <= 0 {
		return nil
	}
	if capacity < bucketSize {
		capacity = bucketSize
	}
	bucketNumber := capacity/bucketSize + 1
	realCapacity := bucketNumber * bucketSize
	// pre-allocate everything to make memory adjacency
	allBuckets := make([]bucket, bucketNumber)
	allPriceLevels := make([]PriceLevel, realCapacity)
	var preBucket *bucket = nil
	for i, j := 0, 0; i < bucketNumber; i++ {
		//TODO: even allocation may not be the most optimised, should try exponential as well
		allBuckets[i].elements = allPriceLevels[j : j+bucketSize]
		allBuckets[i].elements = allBuckets[i].elements[:0]
		j += bucketSize
		if preBucket != nil {
			preBucket.next = &allBuckets[i]
			preBucket = preBucket.next
		} else {
			preBucket = &allBuckets[0]
		}
	}
	//assert preBucket!=nil
	preBucket.next = nil

	return &ULList{
		&allBuckets[0], //at the very beginnig, only one bucket is used
		&allBuckets[1], //assert bucketNumber > 1
		capacity,
		bucketSize,
		comp,
		allBuckets}
}

func (ull *ULList) String() string {
	var buffer bytes.Buffer
	var j int
	for i := ull.begin; i != ull.dend; i = i.next {
		buffer.WriteString(fmt.Sprintf("Bucket %d{", j))
		for _, p := range i.elements {
			buffer.WriteString(fmt.Sprintf("%.8f->[", p.Price))
			for _, o := range p.orders {
				buffer.WriteString(fmt.Sprintf("%s %d %.8f,", o.id, o.time, o.qty))
			}
			buffer.WriteString("]")
		}
		buffer.WriteString("},")
		j++
	}
	return buffer.String()
}

// ensureCapacity() gurantees at least one more free bucket to use,
// otherwise 'double' the size
func (ull *ULList) ensureCapacity() {
	if ull.dend.next == nil { // no empty bucket is available, re-allocate
		oldBucketNumber := ull.capacity/ull.bucketSize + 1
		ull.capacity *= 2
		bucketNumber := ull.capacity/ull.bucketSize + 1
		deltaBucketNumber := bucketNumber - oldBucketNumber
		oldBuckets := ull.allBuckets
		ull.allBuckets = make([]bucket, bucketNumber)
		copy(ull.allBuckets, oldBuckets)
		newPriceLevels := make([]PriceLevel, deltaBucketNumber*ull.bucketSize)
		var preBucket *bucket = ull.dend
		//no need to copy allPriceLevels, since no benefits
		for i, j := oldBucketNumber, int(0); i < bucketNumber; i++ {
			ull.allBuckets[i].elements = newPriceLevels[j : j+ull.bucketSize]
			// clear length
			ull.allBuckets[i].elements = ull.allBuckets[i].elements[:0]
			j += ull.bucketSize
			preBucket.next = &ull.allBuckets[i]
			preBucket = preBucket.next
		}
		preBucket.next = nil
	}
}

//splitBucket() would move one bucket from data end to be after the full
//bucket, and re-allocate half the PriceLevels to it
func (ull *ULList) splitBucket(origin *bucket) *bucket {
	ull.ensureCapacity()
	//assert(ull.dend.next!=nil), i.e. there is still avaiable free bucket
	oldNext := origin.next             //same the next of origin
	origin.next = ull.dend.next        // pick up the one after data end
	ull.dend.next = ull.dend.next.next //shift one after data end
	origin.next.next = oldNext         // re-connect the next of the origin from the new pick up
	oldElements := origin.elements
	newElements := origin.next.elements[:0]
	mid := len(oldElements) / 2
	origin.next.elements = append(newElements, oldElements[mid:]...)
	origin.elements = oldElements[:mid]
	return origin.next
}

func (ull *ULList) Clear() {
	for i := ull.begin; i != ull.dend; i = i.next {
		i.clear()
	}
	ull.dend = ull.begin.next // only leave with one bucket
}

//getBucket return the 'last' bucket which contains price larger-equal (for buy)
//or smaller-equal (for sell) than the input price. If the price is larger (for buy)
//or smaller than any bucket head, nil is returned
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

func (ull *ULList) AddPriceLevel(p *PriceLevel) bool {
	last := ull.getBucket(p.Price)
	if last == nil {
		last = ull.begin
	}
	if last.size() >= ull.bucketSize {
		//bucket is full, split
		//TODO: do we have to wait until it is 100% full?
		next := ull.splitBucket(last)
		if ull.compare(next.head().Price, p.Price) >= 0 {
			return next.insert(p, ull.compare) > 0
		}
		return last.insert(p, ull.compare) > 0
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
			//insert at the data end instead of the final end, so it is closer of the beginning of the memory allocation
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

//GetPriceLevel returns the PriceLevel point that has the same price as p.
//It will return nil if no such price.
func (ull *ULList) GetPriceLevel(p float64) *PriceLevel {
	for i := ull.begin; i != ull.dend; i = i.next {
		h := i.head()
		if h != nil {
			c := ull.compare(h.Price, p)
			switch c {
			case 0: //head has the price
				return h
			case 1: //head is larger (for buy, less for sell)
				if i.next == ull.dend { // last bucket
					return i.get(p, ull.compare)
				}
				h = i.next.head()
				// next bucket is more
				if h != nil {
					switch ull.compare(h.Price, p) {
					case 0:
						return h
					case -1:
						return i.get(p, ull.compare)
					}
				}
				//continue to move to the next bucket
			case -1:
				return nil
			}
		} else { // no way to reach here either
			return nil
		}
	}
	return nil
}
