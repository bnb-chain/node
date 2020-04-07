package order

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
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

	assert := assert.New(t)

	expected := []*SymbolWithOrderNumber{A3, A4, A5}
	result := findTopKLargest([]*SymbolWithOrderNumber{A1, A2, A3, A4, A5}, 3)
	assert.Equal(3, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{A3, A4, A5}
	result = findTopKLargest([]*SymbolWithOrderNumber{A5, A3, A3, A1, A4}, 3)
	assert.Equal(3, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{A3, A4, A5}
	result = findTopKLargest([]*SymbolWithOrderNumber{A5, B3, A3, A1, A4}, 3)
	assert.Equal(3, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{A6, A5}
	result = findTopKLargest([]*SymbolWithOrderNumber{A3, A2, A1, A5, A6, A4}, 2)
	assert.Equal(2, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{A6, C4, B5}
	result = findTopKLargest([]*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}, 3)
	assert.Equal(3, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}
	result = findTopKLargest([]*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}, 6)
	assert.Equal(6, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}
	result = findTopKLargest([]*SymbolWithOrderNumber{D3, E2, F1, B5, A6, C4}, 7)
	assert.Equal(6, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}
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
	assert.Equal(3, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{B2, A2, C2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 3)
	assert.Equal(3, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{A2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 1)
	assert.Equal(1, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{A1, A2, B2, C2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 4)
	assert.Equal(4, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}

	expected = []*SymbolWithOrderNumber{A1, A2, B2, C1, C2}
	result = findTopKLargest([]*SymbolWithOrderNumber{A0, A1, A2, B2, C1, C2}, 5)
	assert.Equal(5, len(result))
	for _, x := range result {
		fmt.Printf("%v,", *x)
	}
	fmt.Println("")
	for _, ele := range expected {
		if !contains(result, ele) {
			t.Fatalf("Expected contains %v, but doesn't exist", ele)
		}
	}
}

func contains(s []*SymbolWithOrderNumber, e *SymbolWithOrderNumber) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
