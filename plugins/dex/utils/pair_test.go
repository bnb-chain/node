package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/node/plugins/dex/utils"
)

func TestCalcLotSizeAndCalcTickSize(t *testing.T) {
	var tests = []struct {
		price    int64
		lotSize  int64
		tickSize int64
	}{
		{-1, 1e8, 1},
		{0, 1e8, 1},
		{1e2, 1e8, 1},
		{1e8, 1e5, 1e3},
		{1e17, 1, 1e12},
	}

	for i := 0; i < len(tests); i++ {
		tickSize, lotSize := utils.CalcTickSizeAndLotSize(tests[i].price)
		assert.Equal(t, tests[i].tickSize, tickSize)
		assert.Equal(t, tests[i].lotSize, lotSize)
	}
}
