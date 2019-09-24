module github.com/binance-chain/node

go 1.12

require (
	github.com/Shopify/sarama v1.21.0
	github.com/btcsuite/btcd v0.0.0-20190115013929-ed77733ec07d
	github.com/cosmos/cosmos-sdk v0.25.0
	github.com/deathowl/go-metrics-prometheus v0.0.0-20190530215645-35bace25558f
	github.com/eapache/go-resiliency v1.1.0
	github.com/ethereum/go-ethereum v1.8.21
	github.com/go-kit/kit v0.6.0
	github.com/google/btree v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/golang-lru v0.5.3
	github.com/linkedin/goavro v0.0.0-20180427201934-fa8f6a30176c
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.1
	github.com/sasha-s/go-deadlock v0.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	github.com/tendermint/go-amino v0.15.0
	github.com/tendermint/iavl v0.12.4
	github.com/tendermint/tendermint v0.32.3
	go.uber.org/ratelimit v0.1.0
)

replace (
	github.com/cosmos/cosmos-sdk => github.com/binance-chain/bnc-cosmos-sdk v0.19.1-0.20190924183902-349ffb735ff6
	github.com/tendermint/go-amino => github.com/binance-chain/bnc-go-amino v0.14.1-binance.1
	github.com/tendermint/iavl => github.com/binance-chain/bnc-tendermint-iavl v0.12.0-binance.1
	github.com/tendermint/tendermint => github.com/binance-chain/bnc-tendermint v0.29.1-binance.3.0.20190923114917-479a59a5dbd7
	github.com/zondax/ledger-cosmos-go => github.com/binance-chain/ledger-cosmos-go v0.9.9-binance.3
	golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20180820045704-3764759f34a5
)
