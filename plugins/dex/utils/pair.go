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

func CalcPriceWMA(prices *utils.FixedSizeRing) int64 {
	n := prices.Count()
	elements := prices.Elements()

	weightedSum := float64(0)
	for i, element := range elements {
		tmp := float64(i+1) * float64(element.(int64))
		weightedSum += tmp
	}
	totalWeight := float64(n * (n+1)/2)
	wma := weightedSum / totalWeight
	return int64(wma)
}
