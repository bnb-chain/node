package utils_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/node/plugins/dex/utils"
)

func TestIsExceedMaxNotional(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(true, utils.IsExceedMaxNotional(math.MaxInt64, math.MaxInt64))
	assert.Equal(true, utils.IsExceedMaxNotional(math.MaxInt64/2, math.MaxInt64/2))
	assert.Equal(false, utils.IsExceedMaxNotional(900e16, 1e6))
	assert.Equal(false, utils.IsExceedMaxNotional(900e16, 1e8))
	assert.Equal(true, utils.IsExceedMaxNotional(900e16, 2e8))
	assert.Equal(false, utils.IsExceedMaxNotional(1e6, 900e16))
	assert.Equal(true, utils.IsExceedMaxNotional(900e16, 2e8))
	assert.Equal(true, utils.IsExceedMaxNotional(2e8, 900e16))
	assert.Equal(true, utils.IsExceedMaxNotional(900e16, 1.5e8))
	assert.Equal(true, utils.IsExceedMaxNotional(1.5e8, 900e16))
	assert.Equal(false, utils.IsExceedMaxNotional(1, 1))
}

func TestIsUnderMinNotional(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(false, utils.IsUnderMinNotional(math.MaxInt64, math.MaxInt64))
	assert.Equal(false, utils.IsUnderMinNotional(math.MaxInt64/2, math.MaxInt64/2))
	assert.Equal(false, utils.IsUnderMinNotional(900e16, 1e6))
	assert.Equal(true, utils.IsUnderMinNotional(1, 1))
	assert.Equal(true, utils.IsUnderMinNotional(1, 1e7))
	assert.Equal(false, utils.IsUnderMinNotional(1e8, 1))
	assert.Equal(false, utils.IsUnderMinNotional(1e4, 1e4))
	assert.Equal(false, utils.IsUnderMinNotional(1e9, 1))

}

func BenchmarkIsExceedMaxNotional_BigInt(b *testing.B) {
	isExceedMaxNotional := func(price, qty int64) bool {
		var bi big.Int
		return !bi.Div(bi.Mul(big.NewInt(qty), big.NewInt(price)), big.NewInt(1e8)).IsInt64()
	}
	for i := 0; i < b.N; i++ {
		isExceedMaxNotional(900e16, 1e8)
	}
}

func BenchmarkIsExceedMaxNotional(b *testing.B) {
	for i := 0; i < b.N; i++ {
		utils.IsExceedMaxNotional(900e16, 1.5e8)
	}
}
