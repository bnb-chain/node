package utils

import (
	"math"

	"github.com/binance-chain/node/common/utils"
)

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

// Warning, this wma is not so accurate and can only be used for calculating tick_size/lot_size
func CalcPriceWMA(prices *utils.FixedSizeRing) int64 {
	n := prices.Count()
	if n == 0 {
		return 0
	}
	elements := prices.Elements()
	var weightedSum int64 = 0
	// when calculate the sum, we ignore the last 5 digits as they have no impact on the tick_size calculation.
	for i, element := range elements {
		weightedSum += int64(i+1) * element.(int64) / 1e5
	}
	totalWeight := int64(n * (n + 1) / 2)
	return weightedSum / totalWeight * 1e5
}
