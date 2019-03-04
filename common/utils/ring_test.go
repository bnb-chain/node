package utils

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixedSizeRing_PushAndElements(t *testing.T) {
	q := NewFixedSizedRing(4)
	q.Push(1)
	require.Equal(t, int64(1), q.size)
	require.Equal(t, int64(1), q.tail)
	require.Equal(t, []interface{}{1}, q.Elements())
	require.Equal(t, []interface{}{1, nil, nil, nil}, q.buf)
	q.Push(2)
	q.Push(3)
	q.Push(4)
	require.Equal(t, int64(0), q.tail)
	require.Equal(t, int64(4), q.size)
	require.Equal(t, []interface{}{1,2,3,4}, q.Elements())
	require.Equal(t, []interface{}{1,2,3,4}, q.buf)
	q.Push(5)
	require.Equal(t, int64(1), q.tail)
	require.Equal(t, int64(4), q.size)
	require.Equal(t, []interface{}{2,3,4,5}, q.Elements())
	require.Equal(t, []interface{}{5,2,3,4}, q.buf)
}

// ~0.26 ns for each mod op
func BenchmarkFixedSizeRing_mod(b *testing.B) {
	nums := make([]int64, 10000)
	for i:=0; i<len(nums); i++ {
		nums[i] = rand.Int63()
	}
	for i:=0; i<b.N; i++ {
		for _, num := range nums {
			_  = num % 2000
		}
	}
}

// ~20ns/op
func BenchmarkFixedSizeRing_Push(b *testing.B) {
	q := NewFixedSizedRing(2000)
	for i:=0; i<b.N; i++ {
		q.Push(i)
	}
}

// ~6000ns/op
func BenchmarkFixedSizeRing_FullElements(b *testing.B) {
	q := NewFixedSizedRing(2000)
	for i:=0; i<2000; i++ {
		q.Push(i)
	}
	for i:=0; i<b.N; i++ {
		_ = q.Elements()
	}
}