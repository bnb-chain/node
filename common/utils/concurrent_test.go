package utils_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/binance-chain/node/common/utils"
)

func TestConcurrentExecuteAsync(t *testing.T) {
	var nums = map[int][]int{}
	for i := 0; i < 100; i++ {
		nums[i] = []int{0}
	}

	var sum int64
	numCh := make(chan int, 4)
	producer := func() {
		for i := 0; i < 100; i++ {
			numCh <- i
		}
		close(numCh)
	}
	consumer := func() {
		for num := range numCh {
			nums[num][0] = num
		}
	}
	postConsume := func() {
		for _, numArr := range nums {
			atomic.AddInt64(&sum, int64(numArr[0]))
		}
	}
	utils.ConcurrentExecuteAsync(4, producer, consumer, postConsume)
	require.NotEqual(t, int64(4950), atomic.LoadInt64(&sum))
	time.Sleep(1e6)
	require.Equal(t, int64(4950), atomic.LoadInt64(&sum))
	for num, numArr := range nums {
		require.Equal(t, num, numArr[0])
	}
}

func TestConcurrentExecuteSync(t *testing.T) {
	var nums = map[int][]int{}
	for i := 0; i < 100; i++ {
		nums[i] = []int{0}
	}

	sum := 0
	numCh := make(chan int, 4)
	producer := func() {
		for i := 0; i < 100; i++ {
			numCh <- i
		}
		close(numCh)
	}
	consumer := func() {
		for num := range numCh {
			nums[num][0] = num
		}
	}
	utils.ConcurrentExecuteSync(4, producer, consumer)
	for num, numArr := range nums {
		require.Equal(t, num, numArr[0])
	}
	for _, numArr := range nums {
		sum += numArr[0]
	}
	require.Equal(t, 4950, sum)
}
