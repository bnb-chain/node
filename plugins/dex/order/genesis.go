package order

type TradingGenesis struct {
	ExpireFee          int64 `json:"expire_fee"`
	ExpireFeeNative    int64 `json:"expire_fee_native"`
	IOCExpireFee       int64 `json:"ioc_expire_fee"`
	IOCExpireFeeNative int64 `json:"ioc_expire_fee_native"`
	CancelFee          int64 `json:"cancel_fee"`
	CancelFeeNative    int64 `json:"cancel_fee_native"`
	FeeRate            int64 `json:"fee_rate"`
	FeeRateNative      int64 `json:"fee_rate_native"`
}

var DefaultTradingGenesis = TradingGenesis{
	ExpireFee:          1e5,
	ExpireFeeNative:    2e4,
	IOCExpireFee:       5e4,
	IOCExpireFeeNative: 1e4,
	CancelFee:          1e5,
	CancelFeeNative:    2e4,
	FeeRate:            500,
	FeeRateNative:      250,
}
