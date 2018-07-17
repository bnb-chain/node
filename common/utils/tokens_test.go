package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTradeSymbol2Ccy(t *testing.T) {
	assert := assert.New(t)
	_, _, e := TradeSymbol2Ccy("hello world")
	assert.EqualError(e, "Failed to parse trade symbol into currencies")
	_, _, e = TradeSymbol2Ccy("BNB_")
	assert.EqualError(e, "Failed to parse trade symbol into currencies")
	_, _, e = TradeSymbol2Ccy("_BNB")
	assert.EqualError(e, "Failed to parse trade symbol into currencies")
	_, _, e = TradeSymbol2Ccy("__BNB")
	assert.EqualError(e, "Failed to parse trade symbol into currencies")
	tr, q, e := TradeSymbol2Ccy("XRP_BNB")
	assert.Equal("XRP", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
	tr, q, e = TradeSymbol2Ccy("XRP.B_BNB")
	assert.Equal("XRP.B", tr)
	assert.Equal("BNB", q)
	assert.Nil(e)
}

func TestCcy2TradeSymbol(t *testing.T) {
	assert := assert.New(t)
	p := Ccy2TradeSymbol("hello", "world")
	assert.Equal("hello_world", p)
}
