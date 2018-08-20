package utils

import "math"

// CalcTickSizeAndLotSize calculate TickSize and LotSize
func CalcTickSizeAndLotSize(price int64) (tickSize, lotSize int64) {
	if price <= 0 {
		return 1, 1e8
	}

	priceDigits := int64(math.Floor(math.Log10(float64(price))))
	tickSizeDigits := int64(math.Max(float64(priceDigits-5), 0))
	lotSizeDigits := int64(math.Max(float64(8-tickSizeDigits), 0))

	return int64(math.Pow(10, float64(tickSizeDigits))), int64(math.Pow(10, float64(lotSizeDigits)))
}
