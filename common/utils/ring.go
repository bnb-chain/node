package utils

import "fmt"

// FixedSizeRing is well-designed for the scenario that we always push elements back and never pop elements
// If the ring is full, the newly pushed elements will override the oldest elements.
// not goroutine-safe
type FixedSizeRing struct {
	buf  []interface{}
	tail int64
	size int64
	cap int64
}

func NewFixedSizedRing(cap int64) *FixedSizeRing {
	return &FixedSizeRing{
		buf:  make([]interface{}, cap, cap),
		tail: 0,
		size: 0,
		cap: cap,
	}
}

func (q *FixedSizeRing) IsEmpty() bool {
	return q.size == 0
}

func (q *FixedSizeRing) Count() int64 {
	return q.size
}

func(q *FixedSizeRing) Push(v interface{}) *FixedSizeRing {
	q.buf[q.tail] = v
	q.tail = (q.tail + 1) % q.cap

	if q.size < q.cap {
		q.size++
	}
	return q
}

func (q *FixedSizeRing) Elements() []interface{} {
	if q.size == 0 {
		return []interface{}{}
	}

	result := make([]interface{}, q.size)
	if q.tail == q.size {
		copy(result, q.buf[:q.tail])
	} else if q.tail < q.size {
		copy(result, q.buf[q.tail:])
		copy(result[q.cap-q.tail:], q.buf[:q.tail])
	} else {
		// should not happen here
		panic(fmt.Errorf("tail should not be bigger than size"))
	}
	return result
}

func (q *FixedSizeRing) String() string {
	return fmt.Sprintf("%#v", q)
}

