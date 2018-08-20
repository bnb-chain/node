package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/BiJie/BinanceChain/plugins/dex/utils"
)

func TestCalcLotSizeAndCalcTickSize(t *testing.T) {
	var lotSizeTests = []struct {
		in  int64
		out int64
	}{
		{-1, 1e8},
		{0, 1e8},
		{1e2, 1e8},
		{1e8, 1e5},
		{1e17, 1},
	}

	var tickSizeTests = []struct {
		in  int64
		out int64
	}{
		{-1, 1},
		{0, 1},
		{1e2, 1},
		{1e8, 1e3},
		{1e17, 1e12},
	}

	for i := 0; i < len(lotSizeTests); i++ {
		assert.Equal(t, utils.CalcLotSize(lotSizeTests[i].in), lotSizeTests[i].out)
	}

	for i := 0; i < len(tickSizeTests); i++ {
		assert.Equal(t, utils.CalcTickSize(tickSizeTests[i].in), tickSizeTests[i].out)
	}
}
