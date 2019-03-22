package utils

import (
	"fmt"
	"math"
	"math/big"
	"strings"

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

// Warning! this wma is not so accurate and can only be used for calculating tick_size/lot_size
// assume the len(prices) is between 500 and 2000
func CalcPriceWMA(prices *utils.FixedSizeRing) int64 {
	n := prices.Count()
	if n == 0 {
		return 0
	}
	elements := prices.Elements()
	var weightedSum int64 = 0
	totalWeight := int64(n * (n + 1) / 2)

	// when calculate the sum, the last 5 digits of price has no impact on the tick_size calculation.
	// so when calc the PWA, we can ignore the last x digits so that in most cases we can use int64 intermediately.
	// diff <= (10^x-1) * n * 10^x / ((n+1)*n/2),
	// assume 500<= n <= 2000, if we let diff < 10^6, then x <= 4

	i, lenPrices := 0, len(elements)
	for ; i<lenPrices; i++ {
		weightedSum += int64(float64(i+1) * float64(elements[i].(int64)) / 1e4)
		if weightedSum < 0 {
			bigWeightedSum := big.NewInt(weightedSum)
			for i++; i<lenPrices; i++ {
				bigWeightedSum.Add(bigWeightedSum, big.NewInt(int64(float64(i+1) * float64(elements[i].(int64)) / 1e4)))
			}
			// res won't overflow
			var res big.Int
			return res.Quo(res.Mul(bigWeightedSum,  big.NewInt(1e4)), big.NewInt(totalWeight)).Int64()
		}
	}

	if weightedSum > 9e14 {
		// res won't overflow
		var res big.Int
		return res.Quo(res.Mul(big.NewInt(weightedSum), big.NewInt(1e4)), big.NewInt(totalWeight)).Int64()
	}
	return weightedSum * 1e4 / totalWeight
}

const DELIMITER = "_"

func TradingPair2Assets(symbol string) (baseAsset, quoteAsset string, err error) {
	assets := strings.SplitN(symbol, DELIMITER, 2)
	if len(assets) != 2 || assets[0] == "" || assets[1] == "" {
		return symbol, "", fmt.Errorf("Failed to parse trading pair symbol:%s into assets", symbol)
	}
	if strings.Contains(assets[1], DELIMITER) {
		return symbol, "", fmt.Errorf("Failed to parse trading pair symbol:%s into assets", symbol)
	}
	return assets[0], assets[1], nil
}

func TradingPair2AssetsSafe(symbol string) (baseAsset, quoteAsset string) {
	baseAsset, quoteAsset, err := TradingPair2Assets(symbol)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse trading pair symbol:%s into assets", symbol))
	}
	return
}

func Assets2TradingPair(baseAsset, quoteAsset string) (symbol string) {
	return fmt.Sprintf("%s_%s", baseAsset, quoteAsset)
}
