package order

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/common"
)

func Test_findKthLargest(t *testing.T) {
	//A0 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 0}
	A1 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 1}
	A2 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 2}
	A3 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 3}
	A4 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 4}
	A5 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 5}
	A6 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 6}
	//B2 := &SymbolWithOrderNumber{"b", 2}
	B3 := &SymbolWithOrderNumber{"BAC-678M_BNB", 3}
	B5 := &SymbolWithOrderNumber{"BAC-678M_BNB", 5}
	//C3 := &SymbolWithOrderNumber{"c", 3}
	C4 := &SymbolWithOrderNumber{"CUY-G42M_BNB", 4}
	//D4 := &SymbolWithOrderNumber{"d", 4}
	D3 := &SymbolWithOrderNumber{"DUY-765_BNB", 3}
	//E5 := &SymbolWithOrderNumber{"e", 5}
	E2 := &SymbolWithOrderNumber{"ETF-876_BNB", 2}
	//F6 := &SymbolWithOrderNumber{"f", 6}
	F1 := &SymbolWithOrderNumber{"FXM-987M_BNB", 1}

	expected := []*SymbolWithOrderNumber{A3, A4, A5}
	result := findTopKLargest([]*SymbolWithOrderNumber{A1, A2, A3, A4, A5}, 3)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{A3, A4, A5}
	result = findTopKLargest([]*SymbolWithOrderNumber{A5, A3, A3, A1, A4}, 3)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{A3, A4, A5}
	result = findTopKLargest([]*SymbolWithOrderNumber{A5, B3, A3, A1, A4}, 3)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{A6, A5}
	result = findTopKLargest([]*SymbolWithOrderNumber{A3, A2, A1, A5, A6, A4}, 2)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{A6, C4, B5}
	result = findTopKLargest([]*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}, 3)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}
	result = findTopKLargest([]*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}, 6)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}
	result = findTopKLargest([]*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}, 7)
	assertResult(t, expected, result)
}

func Test_findKthLargest_SameNumber(t *testing.T) {
	A0 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 0}
	A1 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 1}
	A2 := &SymbolWithOrderNumber{"ABC-9UEM_BNB", 2}
	B2 := &SymbolWithOrderNumber{"BAC-678M_BNB", 2}
	C1 := &SymbolWithOrderNumber{"CUY-G42M_BNB", 1}
	C2 := &SymbolWithOrderNumber{"CUY-G42M_BNB", 2}
	E2 := &SymbolWithOrderNumber{"ETF-876_BNB", 2}
	F2 := &SymbolWithOrderNumber{"FXM-987M_BNB", 2}

	assert := assert.New(t)

	expected := []*SymbolWithOrderNumber{A2, B2, E2}
	result := findTopKLargest([]*SymbolWithOrderNumber{F2, E2, A2, B2, C1}, 3)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{B2, A2, C2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 3)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{A2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 1)
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{A1, A2, B2, C2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 4)
	assert.Equal(4, len(result))
	assertResult(t, expected, result)

	expected = []*SymbolWithOrderNumber{A1, A2, B2, C1, C2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 5)
	assert.Equal(5, len(result))
	assertResult(t, expected, result)
}

func Benchmark_findTopKLargest(b *testing.B) {
	const size = 10000
	origin := make([]*SymbolWithOrderNumber, size)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for i := 0; i < size; i++ {
			origin[i] = &SymbolWithOrderNumber{symbol: string(common.RandBytes(10)), numberOfOrders: int(common.RandIntn(size / 10))}
		}
		b.StartTimer()
		findTopKLargest(origin, size/2)
	}
}

func assertResult(t *testing.T, expected []*SymbolWithOrderNumber, actual []*SymbolWithOrderNumber) {
	var s string
	for _, x := range actual {
		s += fmt.Sprintf("%v,", *x)
	}
	t.Logf(s)
	require.Equal(t, len(expected), len(actual))
	for _, ele := range expected {
		require.Contains(t, actual, ele, "Expected contains %v, but doesn't exist", ele)
	}
}
