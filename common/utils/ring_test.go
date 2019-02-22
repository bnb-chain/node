package utils

import (
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


func BenchmarkFixedSizeRing_Push(b *testing.B) {
	q := NewFixedSizedRing(2000)
	for i:=0; i<b.N; i++ {
		q.Push(i)
	}
}

func BenchmarkFixedSizeRing_FullElements(b *testing.B) {
	q := NewFixedSizedRing(2000)
	for i:=0; i<2000; i++ {
		q.Push(i)
	}
	for i:=0; i<b.N; i++ {
		_ = q.Elements()
	}
}