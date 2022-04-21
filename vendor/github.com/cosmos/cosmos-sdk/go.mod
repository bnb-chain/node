module github.com/cosmos/cosmos-sdk

go 1.16

require (
	github.com/bartekn/go-bip39 v0.0.0-20171116152956-a05967ea095d
	github.com/bgentry/speakeasy v0.1.0
	github.com/binance-chain/tss v0.1.2
	github.com/binance-chain/tss-lib v1.0.0
	github.com/btcsuite/btcd v0.20.0-beta
	github.com/cosmos/go-bip39 v0.0.0-20180819234021-555e2067c45d
	github.com/go-kit/kit v0.9.0
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/golang-lru v0.5.3
	github.com/ipfs/go-log v0.0.1
	github.com/mattn/go-isatty v0.0.10
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pelletier/go-toml v1.4.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/rakyll/statik v0.1.5
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v1.0.1-0.20190318030020-c3a204f8e965
	github.com/tendermint/btcd v0.1.1
	github.com/tendermint/go-amino v0.15.0
	github.com/tendermint/iavl v0.12.4
	github.com/tendermint/tendermint v0.32.3
	github.com/zondax/ledger-cosmos-go v0.9.9
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
)

replace (
	github.com/tendermint/go-amino => github.com/binance-chain/bnc-go-amino v0.14.1-binance.2
	github.com/tendermint/iavl => github.com/binance-chain/bnc-tendermint-iavl v0.12.0-binance.4
	github.com/tendermint/tendermint => github.com/binance-chain/bnc-tendermint v0.32.3-binance.6
	github.com/zondax/ledger-cosmos-go => github.com/binance-chain/ledger-cosmos-go v0.9.9-binance.3
	golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20190823183015-45b1026d81ae
)
