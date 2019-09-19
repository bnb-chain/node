module github.com/binance-chain/node

go 1.12

require (
	github.com/DataDog/zstd v1.4.1
	github.com/Shopify/sarama v1.23.1
	github.com/bartekn/go-bip39 v0.0.0-20171116152956-a05967ea095d
	github.com/beorn7/perks v1.0.1
	github.com/bgentry/speakeasy v0.1.0
	github.com/btcsuite/btcd v0.0.0-20190115013929-ed77733ec07d
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/cosmos/go-bip39 v0.0.0-20180618194314-52158e4697b8
	github.com/cosmos/ledger-go v0.9.2
	github.com/davecgh/go-spew v1.1.1
	github.com/deathowl/go-metrics-prometheus v0.0.0-20190530215645-35bace25558f
	github.com/eapache/go-resiliency v1.2.0
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21
	github.com/eapache/queue v1.1.0
	github.com/etcd-io/bbolt v1.3.3
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-kit/kit v0.9.0
	github.com/go-logfmt/logfmt v0.4.0
	github.com/gogo/protobuf v1.3.0
	github.com/golang/go v0.0.0-20190917043746-c3c53661ba88
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.1
	github.com/google/btree v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/hashicorp/go-uuid v1.0.1
	github.com/hashicorp/golang-lru v0.5.3
	github.com/hashicorp/hcl v1.0.0
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/jcmturner/gofork v1.0.0
	github.com/jmhodges/levigo v1.0.0
	github.com/kr/logfmt v0.0.0-20140226030751-b84e30acd515
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/linkedin/goavro v0.0.0-20190712171002-6315d3704248
	github.com/magiconair/properties v1.8.1
	github.com/mattn/go-isatty v0.0.9
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/natefinch/lumberjack v0.0.0-20170531160350-a96e63847dc3
	github.com/pelletier/go-toml v1.4.0
	github.com/petar/GoLLRB v0.0.0-20190514000832-33fb24c13b99
	github.com/pierrec/lz4 v2.3.0+incompatible
	github.com/pkg/errors v0.8.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/prometheus/procfs v0.0.5
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/rs/cors v1.7.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.0.3
	github.com/stretchr/testify v1.2.1
	github.com/syndtr/goleveldb v1.0.0
	github.com/tendermint/btcd v0.1.1
	github.com/zondax/hid v0.9.0
	go.uber.org/ratelimit v0.1.0
	golang.org/x/net v0.0.0-20190916140828-c8589233b77d
	golang.org/x/sys v0.0.0-20190916202348-b4ddaad3f8a3
	golang.org/x/text v0.3.2
	google.golang.org/genproto v0.0.0-20190916214212-f660b8655731
	google.golang.org/grpc v1.23.1
	gopkg.in/jcmturner/aescts.v1 v1.0.1
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1
	gopkg.in/jcmturner/gokrb5.v7 v7.3.0
	gopkg.in/jcmturner/rpc.v1 v1.1.0
	gopkg.in/yaml.v2 v2.2.2
	github.com/cosmos/ledger-cosmos-go v0.9.9
	github.com/tendermint/go-amino v0.14.1
	github.com/tendermint/iavl v0.12.0
	github.com/tendermint/tendermint v0.32.3
	github.com/cosmos/cosmos-sdk v0.25.0
)

replace (
    github.com/cosmos/ledger-cosmos-go => github.com/binance-chain/ledger-cosmos-go v0.9.9-binance.2
    github.com/tendermint/go-amino => github.com/binance-chain/bnc-go-amino v0.14.1-binance.1
    github.com/tendermint/iavl => github.com/binance-chain/bnc-tendermint-iavl v0.12.0-binance.1
    github.com/tendermint/tendermint => github.com/binance-chain/bnc-tendermint upgrade_0.32.3
    github.com/cosmos/cosmos-sdk => github.com/binance-chain/bnc-cosmos-sdk tendermint-upgrade
)
