package utils_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/BiJie/BinanceChain/common/utils"
)

type Fixed8 = utils.Fixed8

var NewFixed8 = utils.NewFixed8
var Fixed8DecodeString = utils.Fixed8DecodeString

var decimals = utils.Fixed8Decimals

func TestFixed8FromInt64(t *testing.T) {
	value := 1e8   // 1.00000000
	value2 := 1000 // 0.00001000
	assert.Equal(t, Fixed8(value), NewFixed8(1))
	value2s, err := Fixed8DecodeString("0.00001")
	assert.Nil(t, err)
	assert.Equal(t, Fixed8(value2), value2s)
}

func TestNewFixed8(t *testing.T) {
	values := []int64{9000, 1e8, 5, 10945}
	for _, val := range values {
		assert.Equal(t, Fixed8(val*int64(decimals)), NewFixed8(val))
		assert.Equal(t, int64(val), NewFixed8(val).Value())
		assert.Equal(t, int64(val*int64(decimals)), NewFixed8(val).ToInt64())
	}
}

func TestFixed8DecodeString(t *testing.T) {
	// Fixed8DecodeString works correctly with integers
	ivalues := []string{"9000", "100000000", "5", "10945"}
	for _, val := range ivalues {
		n, err := Fixed8DecodeString(val)
		assert.Nil(t, err)
		assert.Equal(t, val+".00000000", n.String())
	}

	// Fixed8DecodeString parses number with maximal precision
	val := "123456789.12345678"
	n, err := Fixed8DecodeString(val)
	assert.Nil(t, err)
	assert.Equal(t, Fixed8(12345678912345678), n)

	// Fixed8DecodeString parses number with non-maximal precision
	val = "901.2341"
	n, err = Fixed8DecodeString(val)
	assert.Nil(t, err)
	assert.Equal(t, Fixed8(90123410000), n)
}

func TestFixed8UnmarshalJSON(t *testing.T) {
	flt := float64(123.45)
	fls := "123.45"
	int := int64(200000000)

	expFltF8, _ := Fixed8DecodeString(fls)
	expIntF8 := NewFixed8(2) // 2.00000000

	// UnmarshalJSON should decode floats
	var fltF8 Fixed8
	s, _ := json.Marshal(flt)
	assert.Nil(t, json.Unmarshal(s, &fltF8))
	assert.Equal(t, expFltF8, fltF8)

	// UnmarshalJSON should decode ints
	var intF8 Fixed8
	s, _ = json.Marshal(int)
	assert.Nil(t, json.Unmarshal(s, &intF8))
	assert.Equal(t, expIntF8, intF8)

	// UnmarshalJSON should decode strings
	var flsF8 Fixed8
	s, _ = json.Marshal(fls)
	assert.Nil(t, json.Unmarshal(s, &flsF8))
	assert.Equal(t, expFltF8, flsF8)
}
