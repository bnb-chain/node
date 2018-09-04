package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTradingPair2Asset(t *testing.T) {
	assert := assert.New(t)
	_, _, e := TradingPair2Assets("hello world")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	_, _, e = TradingPair2Assets("BNB_")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	_, _, e = TradingPair2Assets("_BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	_, _, e = TradingPair2Assets("__BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	tr, q, e := TradingPair2Assets("XRP_BNB")
	assert.Equal("XRP", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
	tr, q, e = TradingPair2Assets("XRP.B_BNB")
	assert.Equal("XRP.B", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
}

func TestAsset2TradingPair(t *testing.T) {
	assert := assert.New(t)
	p := Assets2TradingPair("hello", "world")
	assert.Equal("hello_world", p)
}
