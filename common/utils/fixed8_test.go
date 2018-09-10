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
	fl := float64(123.45)
	str := "123.45"
	expected, _ := Fixed8DecodeString(str)

	// UnmarshalJSON should decode floats
	var u1 Fixed8
	s, _ := json.Marshal(fl)
	assert.Nil(t, json.Unmarshal(s, &u1))
	assert.Equal(t, expected, u1)

	// UnmarshalJSON should decode strings
	var u2 Fixed8
	s, _ = json.Marshal(str)
	assert.Nil(t, json.Unmarshal(s, &u2))
	assert.Equal(t, expected, u2)
}
