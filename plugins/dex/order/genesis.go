package order

type TradingGenesis struct {
	MakerFee             int64 `json:"makerFee"`
	TakerFee             int64 `json:"takerFee"`
	FeeFactor            int64 `json:"feeFactor"`
	MaxFee               int64 `json:"maxFee"`
	NativeTokenDiscount  int64 `json:"nativeTokenDiscount"`
	VolumeBucketDuration int64 `json:"volumeBucketDuration"`
}

// dex fee settings - established in genesis block
// feeFactor: 25 => 0.25% (0.0025)
// maxFee: 50/10000 = 0.5% (0.005)
// nativeTokenDiscount: 1/2 => 50%
// volumeBucketDuration: 82800secs = 23hrs
var DefaultTradingGenesis = TradingGenesis{
	MakerFee:             25,
	TakerFee:             30,
	FeeFactor:            10000,
	MaxFee:               5000,
	NativeTokenDiscount:  2,
	VolumeBucketDuration: 82800,
}
