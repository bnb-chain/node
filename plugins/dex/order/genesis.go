package order

type TradingGenesis struct {
	MakerFee             int64 `json:"makerFee"`
	TakerFee             int64 `json:"takerFee"`
	FeeFactor            int64 `json:"feeFactor"`
	MaxFee               int64 `json:"maxFee"`
	NativeTokenDiscount  int64 `json:"nativeTokenDiscount"`
	VolumeBucketDuration int64 `json:"volumeBucketDuration"`
}
