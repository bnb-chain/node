package types

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcLotSizeAndCalcTickSize(t *testing.T) {
	var lotSizeTests = []struct {
		in  int64
		out int64
	}{
		{-1, 1},
		{0, 1},
		{int64(math.Pow(10, 2)), int64(math.Pow(10, 8))},
		{int64(math.Pow(10, 8)), int64(math.Pow(10, 5))},
		{int64(math.Pow(10, 17)), int64(math.Pow(10, 0))},
	}

	var tickSizeTests = []struct {
		in  int64
		out int64
	}{
		{-1, 1},
		{0, 1},
		{int64(math.Pow(10, 2)), int64(math.Pow(10, 0))},
		{int64(math.Pow(10, 8)), int64(math.Pow(10, 3))},
		{int64(math.Pow(10, 17)), int64(math.Pow(10, 12))},
	}

	for i := 0; i < len(lotSizeTests); i++ {
		assert.Equal(t, CalcLotSize(lotSizeTests[i].in), lotSizeTests[i].out)
	}

	for i := 0; i < len(tickSizeTests); i++ {
		assert.Equal(t, CalcTickSize(tickSizeTests[i].in), tickSizeTests[i].out)
	}
}
