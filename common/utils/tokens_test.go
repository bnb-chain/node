package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTradingPair2Asset(t *testing.T) {
	assert := assert.New(t)
	_, _, e := TradingPair2Asset("hello world")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	_, _, e = TradingPair2Asset("BNB_")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	_, _, e = TradingPair2Asset("_BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	_, _, e = TradingPair2Asset("__BNB")
	assert.EqualError(e, "Failed to parse trading pair symbol into assets")
	tr, q, e := TradingPair2Asset("XRP_BNB")
	assert.Equal("XRP", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
	tr, q, e = TradingPair2Asset("XRP.B_BNB")
	assert.Equal("XRP.B", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
}

func TestAsset2TradingPair(t *testing.T) {
	assert := assert.New(t)
	p := Asset2TradingPair("hello", "world")
	assert.Equal("hello_world", p)
}
