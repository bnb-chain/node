package order

import "github.com/tendermint/tendermint/libs/common"

//Find and return top K symbols with largest number of order.
// The returned top K slice is not sorted. The input orderNums may be re-ordered in place.
// If more than one symbols have same order numbers, these symbol will be selected by ascending alphabetical sequence.
func findTopKLargest(orderNums []*SymbolWithOrderNumber, k int) []*SymbolWithOrderNumber {
	if k >= len(orderNums) {
		return orderNums
	}
	return quickselect(orderNums, 0, len(orderNums)-1, k)
}

func partition(orderNums []*SymbolWithOrderNumber, start, end int) int {
	// move pivot to end
	if end == start {
		return start
	}

	pivot := common.RandIntn(end-start) + start
	orderNums[end], orderNums[pivot] = orderNums[pivot], orderNums[end]
	pivotValue := orderNums[end]
	i := start
	for j := start; j < end; j++ {
		if compare(orderNums[j], pivotValue) {
			orderNums[i], orderNums[j] = orderNums[j], orderNums[i]
			i++
		}
	}
	// move pivot to its sorted position
	orderNums[i], orderNums[end] = orderNums[end], orderNums[i]
	// return pivot index
	return i
}

func compare(orderNumA *SymbolWithOrderNumber, orderNumB *SymbolWithOrderNumber) bool {
	if orderNumA.numberOfOrders > orderNumB.numberOfOrders {
		return true
	} else if orderNumA.numberOfOrders == orderNumB.numberOfOrders {
		return orderNumA.symbol < orderNumB.symbol
	}
	return false
}

func quickselect(orderNums []*SymbolWithOrderNumber, start, end, n int) []*SymbolWithOrderNumber {
	// use last element as pivot
	pivotIndex := partition(orderNums, start, end)

	if n-1 == pivotIndex {
		return orderNums[:pivotIndex+1]
	} else if n-1 > pivotIndex {
		return quickselect(orderNums, pivotIndex+1, end, n)
	} else {
		return quickselect(orderNums, start, pivotIndex-1, n)
	}
}
