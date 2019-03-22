package utils_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmnutils "github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/utils"
)

func TestCalcLotSizeAndCalcTickSize(t *testing.T) {
	var tests = []struct {
		price    int64
		lotSize  int64
		tickSize int64
	}{
		{-1, 1e13, 1},
		{0, 1e13, 1},
		{1e2, 1e11, 1},
		{1e8, 1e5, 1e3},
		{1e17, 1, 1e12},
	}

	for i := 0; i < len(tests); i++ {
		tickSize, lotSize := utils.CalcTickSizeAndLotSize(tests[i].price)
		assert.Equal(t, tests[i].tickSize, tickSize)
		assert.Equal(t, tests[i].lotSize, lotSize)
	}
}

func BenchmarkRecentPrices_Size(b *testing.B) {
	pricesRing := cmnutils.NewFixedSizedRing(2000)
	prices := make([]int64, 2000)
	for i := 0; i < 2000; i++ {
		prices[i] = rand.Int63()
	}
	for i := 0; i < 2000; i++ {
		pricesRing.Push(prices[i])
	}

	recentPrices := make(map[string]*cmnutils.FixedSizeRing, 256)
	for i := 0; i < 10; i++ {
		recentPrices[string(i)] = pricesRing
	}

	bz, _ := json.Marshal(pricesRing.Elements())

	for i := 0; i < b.N; i++ {
		bz, _ = cmnutils.Compress(bz)
	}

}

func TestCalcPriceWMA_Basic(t *testing.T) {
	prices := cmnutils.NewFixedSizedRing(10)
	prices.Push(int64(1e5))
	require.Equal(t, int64(1e5), utils.CalcPriceWMA(prices))
	prices.Push(int64(2e5))
	require.Equal(t, int64(1e5), utils.CalcPriceWMA(prices))
	prices.Push(int64(3e5)).Push(int64(4e5)).Push(int64(5e5)).Push(int64(6e5))
	require.Equal(t, int64(4e5), utils.CalcPriceWMA(prices))
}

func TestCalcPriceWMA_Real(t *testing.T) {
	for k := 0; k < 2000; k++ {
		prices := make([]int64, 2000)
		for i := 0; i < 2000; i++ {
			prices[i] = int64((i + 1) * 1e8)
		}
		pricesRing := cmnutils.NewFixedSizedRing(2000)
		for i := 0; i < 2000; i++ {
			pricesRing.Push(prices[i])
		}
		require.Equal(t, int64(133366600000), utils.CalcPriceWMA(pricesRing))
	}
}

// about 8800 ns/op for 2000 prices, including some FixedSizedRing ops.
func BenchmarkCalcPriceWMA(b *testing.B) {
	prices := cmnutils.NewFixedSizedRing(2000)
	for i := 0; i < 2000; i++ {
		prices.Push(rand.Int63())
	}

	for i := 0; i < b.N; i++ {
		utils.CalcPriceWMA(prices)
	}
}
