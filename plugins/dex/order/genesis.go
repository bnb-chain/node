package order

type TradingGenesis struct {
	ExpireFee     int64 `json:"expire_fee"`
	IOCExpireFee  int64 `json:"ioc_expire_fee"`
	FeeRateNative int64 `json:"fee_rate_native"`
	FeeRate       int64 `json:"fee_rate"`
}

// TODO: determine the fee/feeRate
var DefaultTradingGenesis = TradingGenesis{
	ExpireFee:     10000,
	IOCExpireFee:  5000,
	FeeRateNative: 500,
	FeeRate:       1000,
}
