package utils

import (
	"fmt"
	"github.com/binance-chain/node/common/types"
	"math"
	"math/big"
	"strings"

	"github.com/binance-chain/node/common/utils"
)

// CalcTickSize & CalcLotSize calculate TickSize and LotSize
// When calculate TickSize, the input price should be the price of the pair
// When calculate LotSize, the input price should be the price of the base asset against NativeToken

//Price		≥1e8	≥1e7	≥1e6	≥1e5	≥1e4	≥1e3	≥1e2	≥1e1	≥1
//TickSize	1e3		1e2		1e1		1		1		1		1		1		1
//LotSize	1e5		1e6		1e7		1e8		1e9		1e10	1e11	1e12	1e13

//Price		≥1e9	≥1e10	≥1e11	≥1e12	≥1e13	≥1e14	≥1e15	≥1e16	≥1e17
//TickSize	1e4		1e5		1e6		1e7		1e8		1e9		1e10	1e11	1e12
//LotSize	1e4		1e3		1e2		1e1		1		1		1		1		1
func CalcTickSize(price int64) int64 {
	if price <= 0 {
		return 1
	}
	priceDigits := int64(math.Floor(math.Log10(float64(price))))
	tickSizeDigits := int64(math.Max(float64(priceDigits-5), 0))
	return int64(math.Pow(10, float64(tickSizeDigits)))
}

func CalcLotSize(price int64) int64 {
	if price <= 0 {
		return 1e13
	}
	priceDigits := int64(math.Floor(math.Log10(float64(price))))
	lotSizeDigits := int64(math.Max(float64(13-priceDigits), 0))
	return int64(math.Pow(10, float64(lotSizeDigits)))
}

func CalcPriceWMA(prices *utils.FixedSizeRing) int64 {
	n := prices.Count()
	if n == 0 {
		return 0
	}
	elements := prices.Elements()
	totalWeight := int64(n * (n + 1) / 2)

	weightedSum := big.NewInt(0)
	lenPrices := len(elements)
	for i := 0; i < lenPrices; i++ {
		var weightedPrice big.Int
		weightedPrice.Mul(big.NewInt(int64(i+1)), big.NewInt(elements[i].(int64)))
		weightedSum.Add(weightedSum, &weightedPrice)
	}

	// res won't overflow
	var res big.Int
	return res.Quo(weightedSum, big.NewInt(totalWeight)).Int64()
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

func IsMiniTokenTradingPair(symbol string) bool {
	baseAsset, _, err := TradingPair2Assets(symbol)
	if err != nil{
		return false
	}
	return types.IsMiniTokenSymbol(baseAsset)
}
