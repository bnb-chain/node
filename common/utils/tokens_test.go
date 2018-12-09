package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTradingPair2Asset(t *testing.T) {
	assert := assert.New(t)
	_, _, e := TradingPair2Assets("hello world")
	assert.EqualError(e, "Failed to parse trading pair symbol:hello world into assets")
	_, _, e = TradingPair2Assets("BNB_")
	assert.EqualError(e, "Failed to parse trading pair symbol:BNB_ into assets")
	_, _, e = TradingPair2Assets("_BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol:_BNB into assets")
	_, _, e = TradingPair2Assets("__BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol:__BNB into assets")
	_, _, e = TradingPair2Assets("BNB_ABC_XYZ")
	assert.EqualError(e, "Failed to parse trading pair symbol:BNB_ABC_XYZ into assets")
	tr, q, e := TradingPair2Assets("XRP_BNB")
	assert.Equal("XRP", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
	tr, q, e = TradingPair2Assets("XRP.B_BNB")
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
			TradingPair2AssetsSafe(symbol)
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
			ba, qa := TradingPair2AssetsSafe(symbol)
			assert.Equal(ba, expectedBa)
			assert.Equal(qa, expectedQa)
		}()
	}
	wg.Wait()
}

func TestAsset2TradingPair(t *testing.T) {
	assert := assert.New(t)
	p := Assets2TradingPair("hello", "world")
	assert.Equal("hello_world", p)
}
