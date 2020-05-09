module github.com/binance-chain/node

go 1.12

require (
	github.com/Shopify/sarama v1.21.0
	github.com/cosmos/cosmos-sdk v0.25.0
	github.com/deathowl/go-metrics-prometheus v0.0.0-20170731161557-091131e49c33
	github.com/eapache/go-resiliency v1.1.0
	github.com/go-kit/kit v0.9.0
	github.com/google/btree v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/golang-lru v0.5.3
	github.com/linkedin/goavro v0.0.0-20180427201934-fa8f6a30176c
	github.com/mitchellh/go-homedir v1.1.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v1.0.1-0.20190625080220-02440ea7a285 // indirect
	github.com/tendermint/go-amino v0.15.0
	github.com/tendermint/iavl v0.12.4
	github.com/tendermint/tendermint v0.32.3
	go.uber.org/ratelimit v0.1.0
	google.golang.org/grpc v1.26.0 // indirect
	gopkg.in/linkedin/goavro.v1 v1.0.5 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace (
	github.com/cosmos/cosmos-sdk => github.com/binance-chain/bnc-cosmos-sdk v0.19.1-0.20200514092159-efe36398a0b5
	github.com/tendermint/go-amino => github.com/binance-chain/bnc-go-amino v0.14.1-binance.2
	github.com/tendermint/iavl => github.com/binance-chain/bnc-tendermint-iavl v0.12.0-binance.3
	github.com/tendermint/tendermint => github.com/binance-chain/bnc-tendermint v0.32.3-binance.1.0.20200430083314-39f228dd6425
	github.com/zondax/ledger-cosmos-go => github.com/binance-chain/ledger-cosmos-go v0.9.9-binance.3
	golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20190823143015-45b1026d81ae
)
