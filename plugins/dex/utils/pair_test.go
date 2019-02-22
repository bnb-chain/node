package utils_test

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmnutils "github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/utils"
)

func calcWMA_bigInt(prices []int64) int64 {
	n := len(prices)
	totalWeight := int64(n * (n+1)/2)
	var wma big.Int
	var weightedSum big.Int
	var tmp big.Int
	for i, price := range prices {
		tmp.Mul(big.NewInt(int64(i+1)), big.NewInt(price))
		weightedSum.Add(&weightedSum, &tmp)
	}

	wma.Quo(&weightedSum, big.NewInt(totalWeight))
	return wma.Int64()
}

func TestCalcLotSizeAndCalcTickSize(t *testing.T) {
	var tests = []struct {
		price    int64
		lotSize  int64
		tickSize int64
	}{
		{-1, 1e8, 1},
		{0, 1e8, 1},
		{1e2, 1e8, 1},
		{1e8, 1e5, 1e3},
		{1e17, 1, 1e12},
	}

	for i := 0; i < len(tests); i++ {
		tickSize, lotSize := utils.CalcTickSizeAndLotSize(tests[i].price)
		assert.Equal(t, tests[i].tickSize, tickSize)
		assert.Equal(t, tests[i].lotSize, lotSize)
	}
}

func TestCalcPriceWMA_Basic(t *testing.T) {
	prices := cmnutils.NewFixedSizedRing(10)
	prices.Push(int64(1))
	require.Equal(t, int64(1), utils.CalcPriceWMA(prices))
	prices.Push(int64(2))
	require.Equal(t, int64(1), utils.CalcPriceWMA(prices))
	prices.Push(int64(3))
	prices.Push(int64(4))
	prices.Push(int64(5))
	prices.Push(int64(6))
	require.Equal(t, int64(4), utils.CalcPriceWMA(prices))
}

func TestCalcPriceWMA_Real(t *testing.T) {
	for k:=0; k<2000; k++ {
		prices := make([]int64, 2000)
		for i:=0; i<2000; i++ {
			prices[i] = rand.Int63()
		}
		pricesRing := cmnutils.NewFixedSizedRing(2000)
		for i:= 0; i<2000; i++ {
			pricesRing.Push(prices[i])
		}
		require.Equal(t, calcWMA_bigInt(prices), utils.CalcPriceWMA(pricesRing))
	}
}

// about 9000ns/op for 2000 prices, including some FixedSizedRing ops wasted
func BenchmarkCalcPriceWMA(b *testing.B) {
	prices := cmnutils.NewFixedSizedRing(2000)
	for i:=0; i<2000; i++ {
		prices.Push(rand.Int63())
	}

	for i:=0; i<b.N; i++ {
		utils.CalcPriceWMA(prices)
	}
}

// about 300000 ns/op for 2000 prices, need to verify the result from CalcPriceWMA
func BenchmarkWMA_bigInt(b *testing.B) {
	prices := make([]int64, 2000)
	for i:=0; i<2000; i++ {
		prices[i] = rand.Int63()
	}

	for i:=0; i<b.N; i++ {
		calcWMA_bigInt(prices)
	}
}