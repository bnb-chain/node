package utils_test

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"sync"
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
		tickSize := utils.CalcTickSize(tests[i].price)
		assert.Equal(t, tests[i].tickSize, tickSize)
		lotSize := utils.CalcLotSize(tests[i].price)
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
		recentPrices[strconv.Itoa(i)] = pricesRing
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
	require.Equal(t, int64(166666), utils.CalcPriceWMA(prices))
	prices.Push(int64(3e5)).Push(int64(4e5)).Push(int64(5e5)).Push(int64(6e5))
	require.Equal(t, int64(433333), utils.CalcPriceWMA(prices))
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
		require.Equal(t, int64(133366666666), utils.CalcPriceWMA(pricesRing))
	}
}

// about 9000 ns/op for 2000 prices, including some FixedSizedRing ops.
func BenchmarkCalcPriceWMA_SmallPrice(b *testing.B) {
	prices := cmnutils.NewFixedSizedRing(2000)
	for i := 0; i < 2000; i++ {
		prices.Push(rand.Int63n(100e8))
	}

	for i := 0; i < b.N; i++ {
		utils.CalcPriceWMA(prices)
	}
}

// about 160000 ns/op for 2000 prices, including some FixedSizedRing ops.
func BenchmarkCalcPriceWMA_BigPrice(b *testing.B) {
	prices := cmnutils.NewFixedSizedRing(2000)
	for i := 0; i < 2000; i++ {
		prices.Push(rand.Int63())
	}

	for i := 0; i < b.N; i++ {
		utils.CalcPriceWMA(prices)
	}
}

func TestTradingPair2Asset(t *testing.T) {
	assert := assert.New(t)
	_, _, e := utils.TradingPair2Assets("hello world")
	assert.EqualError(e, "Failed to parse trading pair symbol:hello world into assets")
	_, _, e = utils.TradingPair2Assets("BNB_")
	assert.EqualError(e, "Failed to parse trading pair symbol:BNB_ into assets")
	_, _, e = utils.TradingPair2Assets("_BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol:_BNB into assets")
	_, _, e = utils.TradingPair2Assets("__BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol:__BNB into assets")
	_, _, e = utils.TradingPair2Assets("BNB_ABC_XYZ")
	assert.EqualError(e, "Failed to parse trading pair symbol:BNB_ABC_XYZ into assets")
	tr, q, e := utils.TradingPair2Assets("XRP_BNB")
	assert.Equal("XRP", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
	tr, q, e = utils.TradingPair2Assets("XRP.B_BNB")
	assert.Equal("XRP.B", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
}

func TestAsset2TradingPairSafe(t *testing.T) {
	// Test invalid
	var invalidSymbols = []string{"hello world", "BNB_", "__BNB", "_BNB"}
	wg := sync.WaitGroup{}
	wg.Add(len(invalidSymbols))
	for i := range invalidSymbols {
		symbol := invalidSymbols[i]
		go func() {
			defer func(inerSymbol string) {
				if r := recover(); r == nil {
					t.Errorf("Parse trading pair symbol: %s do not panic in Asset2TradingPairSafe", inerSymbol)
				}
				wg.Done()
			}(symbol)
			utils.TradingPair2AssetsSafe(symbol)
		}()
	}
	wg.Wait()

	// Test valid
	var validSymbols = []string{"XRP_BNB", "XRP.B_BNB"}
	var validBaseAsserts = []string{"XRP", "XRP.B"}
	var validQuotaAsserts = []string{"BNB", "BNB"}
	wg = sync.WaitGroup{}
	wg.Add(len(validSymbols))
	assert := assert.New(t)
	for i := range validSymbols {
		symbol := validSymbols[i]
		expectedBa := validBaseAsserts[i]
		expectedQa := validQuotaAsserts[i]
		go func() {
			defer func(inerSymbol string) {
				if r := recover(); r != nil {
					t.Errorf("Parse trading pair symbol: %s do panic in Asset2TradingPairSafe", inerSymbol)
				}
				wg.Done()
			}(symbol)
			ba, qa := utils.TradingPair2AssetsSafe(symbol)
			assert.Equal(ba, expectedBa)
			assert.Equal(qa, expectedQa)
		}()
	}
	wg.Wait()
}

func TestAsset2TradingPair(t *testing.T) {
	assert := assert.New(t)
	p := utils.Assets2TradingPair("hello", "world")
	assert.Equal("hello_world", p)
}
