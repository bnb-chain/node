package utils

import (
	"math"

	"github.com/binance-chain/node/common/utils"
)

// CalcTickSizeAndLotSize calculate TickSize and LotSize

//Price		≥1e8	≥1e7	≥1e6	≥1e5	≥1e4	≥1e3	≥1e2	≥1e1	≥1
//TickSize	1e3		1e2		1e1		1		1		1		1		1		1
//LotSize	1e5		1e6		1e7		1e8		1e9		1e10	1e11	1e12	1e13

//Price		≥1e9	≥1e10	≥1e11	≥1e12	≥1e13	≥1e14	≥1e15	≥1e16	≥1e17
//TickSize	1e4		1e5		1e6		1e7		1e8		1e9		1e10	1e11	1e12
//LotSize	1e4		1e3		1e2		1e1		1		1		1		1		1
func CalcTickSizeAndLotSize(price int64) (tickSize, lotSize int64) {
	if price <= 0 {
		return 1, 1e13
	}

	priceDigits := int64(math.Floor(math.Log10(float64(price))))
	tickSizeDigits := int64(math.Max(float64(priceDigits-5), 0))
	lotSizeDigits := int64(math.Max(float64(13-priceDigits), 0))

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
