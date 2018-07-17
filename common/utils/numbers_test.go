package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbsInt(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(int64(9999999999999999), AbsInt(-9999999999999999))
	assert.Equal(int64(0), AbsInt(0))
	assert.Equal(int64(9999999999999999), AbsInt(9999999999999999))
}

func TestMinInt(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(int64(999999999999), MinInt(999999999999, 999999999999))
	assert.Equal(int64(-999999999999), MinInt(-999999999999, 999999999999))
	assert.Equal(int64(0), MinInt(999999999999, 0))
	assert.Equal(int64(3), MinInt(999999999999, 3))
}

func TestMaxInt(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(int64(999999999999), MaxInt(999999999999, 999999999999))
	assert.Equal(int64(999999999999), MaxInt(-999999999999, 999999999999))
	assert.Equal(int64(0), MaxInt(-999999999999, 0))
	assert.Equal(int64(999999999999), MaxInt(999999999999, 0))
	assert.Equal(int64(3), MaxInt(1, 3))
}
