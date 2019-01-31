package config

import (
	"bytes"
	"path/filepath"
	"text/template"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/common"
)

var configTemplate *template.Template

func init() {
	var err error
	if configTemplate, err = template.New("configFileTemplate").Parse(appConfigTemplate); err != nil {
		panic(err)
	}
}

const (
	AppConfigFileName = "app"
)

// Note: any changes to the comments/variables/mapstructure
// must be reflected in the appropriate struct in config/config.go
const appConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

[base]
# Interval blocks of breathe block, if breatheBlockInterval is 0, breathe block will be created every day.
breatheBlockInterval = {{ .BaseConfig.BreatheBlockInterval }}
# Size of account cache
accountCacheSize = {{ .BaseConfig.AccountCacheSize }}
# Size of signature cache
signatureCacheSize = {{ .BaseConfig.SignatureCacheSize }}
# Running mode when start up, 0: Normal, 1: TransferOnly, 2: RecoverOnly
startMode = {{ .BaseConfig.StartMode }}
# Concurrency of OrderKeeper, should be power of 2
orderKeeperConcurrency = {{ .BaseConfig.OrderKeeperConcurrency }}
# Days count back for breathe block
breatheBlockDaysCountBack = {{ .BaseConfig.BreatheBlockDaysCountBack }}

[addr]
# Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
bech32PrefixAccAddr = "{{ .AddressConfig.Bech32PrefixAccAddr }}"
# Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
bech32PrefixAccPub = "{{ .AddressConfig.Bech32PrefixAccPub }}"
# Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
bech32PrefixValAddr = "{{ .AddressConfig.Bech32PrefixValAddr }}"
# Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
bech32PrefixValPub = "{{ .AddressConfig.Bech32PrefixValPub }}"
# Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
bech32PrefixConsAddr = "{{ .AddressConfig.Bech32PrefixConsAddr }}"
# Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
bech32PrefixConsPub = "{{ .AddressConfig.Bech32PrefixConsPub }}"

##### publication related configurations #####
[publication]
# configurations ends with Kafka can be a semi-colon separated host-port list
# Whether we want publish market data (this includes trades and order)
publishOrderUpdates = {{ .PublicationConfig.PublishOrderUpdates }}
orderUpdatesTopic = "{{ .PublicationConfig.OrderUpdatesTopic }}"
orderUpdatesKafka = "{{ .PublicationConfig.OrderUpdatesKafka }}"

# Whether we want publish account balance to notify browser db indexer persist latest account balance change
publishAccountBalance = {{ .PublicationConfig.PublishAccountBalance }}
accountBalanceTopic = "{{ .PublicationConfig.AccountBalanceTopic }}"
accountBalanceKafka = "{{ .PublicationConfig.AccountBalanceKafka }}"

# Whether we want publish order book changes
publishOrderBook = {{ .PublicationConfig.PublishOrderBook }}
orderBookTopic = "{{ .PublicationConfig.OrderBookTopic }}"
orderBookKafka = "{{ .PublicationConfig.OrderBookKafka }}"

# Whether we want publish block fee changes
publishBlockFee = {{ .PublicationConfig.PublishBlockFee }}
blockFeeTopic = "{{ .PublicationConfig.BlockFeeTopic }}"
blockFeeKafka = "{{ .PublicationConfig.BlockFeeKafka }}"

# Whether we want publish transfers
publishTransfer = {{ .PublicationConfig.PublishTransfer }}
transferTopic = "{{ .PublicationConfig.TransferTopic }}"
transferKafka = "{{ .PublicationConfig.TransferKafka }}"

# Global setting
publicationChannelSize = "{{ .PublicationConfig.PublicationChannelSize }}"
publishKafka = {{ .PublicationConfig.PublishKafka }}
publishLocal = {{ .PublicationConfig.PublishLocal }}
# max size in megabytes of marketdata json file before rotate
localMaxSize = {{ .PublicationConfig.LocalMaxSize }}
# max days of marketdata json files to keep before deleted
localMaxAge = {{ .PublicationConfig.LocalMaxAge }}

[log]

# Write logs to console instead of file
logToConsole = {{ .LogConfig.LogToConsole }}

## The below parameters take effect only when logToConsole is false
# Log file root, if not set, use home path
logFileRoot = "{{ .LogConfig.LogFileRoot }}"
# Log file path relative to log file root path
logFilePath = "{{ .LogConfig.LogFilePath }}"
# Number of logs keep in memory before writing to file
logBuffSize = {{ .LogConfig.LogBuffSize }}
`

type BinanceChainContext struct {
	*server.Context
	*viper.Viper
	*BinanceChainConfig
}

func NewDefaultContext() *BinanceChainContext {
	return &BinanceChainContext{
		server.NewDefaultContext(),
		viper.New(),
		DefaultBinanceChainConfig()}
}

func (context *BinanceChainContext) ToCosmosServerCtx() *server.Context {
	return context.Context
}

type BinanceChainConfig struct {
	*AddressConfig     `mapstructure:"addr"`
	*PublicationConfig `mapstructure:"publication"`
	*LogConfig         `mapstructure:"log"`
	*BaseConfig        `mapstructure:"base"`
}

func DefaultBinanceChainConfig() *BinanceChainConfig {
	return &BinanceChainConfig{
		AddressConfig:     defaultAddressConfig(),
		PublicationConfig: defaultPublicationConfig(),
		LogConfig:         defaultLogConfig(),
		BaseConfig:        defaultBaseConfig(),
	}
}

type AddressConfig struct {
	Bech32PrefixAccAddr  string `mapstructure:"bech32PrefixAccAddr"`
	Bech32PrefixAccPub   string `mapstructure:"bech32PrefixAccPub"`
	Bech32PrefixValAddr  string `mapstructure:"bech32PrefixValAddr"`
	Bech32PrefixValPub   string `mapstructure:"bech32PrefixValPub"`
	Bech32PrefixConsAddr string `mapstructure:"bech32PrefixConsAddr"`
	Bech32PrefixConsPub  string `mapstructure:"bech32PrefixConsPub"`
}

func defaultAddressConfig() *AddressConfig {
	return &AddressConfig{
		Bech32PrefixAccAddr:  "bnb",
		Bech32PrefixAccPub:   "bnbp",
		Bech32PrefixValAddr:  "bva",
		Bech32PrefixValPub:   "bvap",
		Bech32PrefixConsAddr: "bca",
		Bech32PrefixConsPub:  "bcap",
	}
}

type PublicationConfig struct {
	PublishOrderUpdates bool   `mapstructure:"publishOrderUpdates"`
	OrderUpdatesTopic   string `mapstructure:"orderUpdatesTopic"`
	OrderUpdatesKafka   string `mapstructure:"orderUpdatesKafka"`

	PublishAccountBalance bool   `mapstructure:"publishAccountBalance"`
	AccountBalanceTopic   string `mapstructure:"accountBalanceTopic"`
	AccountBalanceKafka   string `mapstructure:"accountBalanceKafka"`

	PublishOrderBook bool   `mapstructure:"publishOrderBook"`
	OrderBookTopic   string `mapstructure:"orderBookTopic"`
	OrderBookKafka   string `mapstructure:"orderBookKafka"`

	PublishBlockFee bool   `mapstructure:"publishBlockFee"`
	BlockFeeTopic   string `mapstructure:"blockFeeTopic"`
	BlockFeeKafka   string `mapstructure:"blockFeeKafka"`

	PublishTransfer bool   `mapstructure:"publishTransfer"`
	TransferTopic   string `mapstructure:"transferTopic"`
	TransferKafka   string `mapstructure:"transferKafka"`

	PublicationChannelSize int `mapstructure:"publicationChannelSize"`

	// DO NOT put this option in config file
	// deliberately make it only a command line arguments
	// https://github.com/binance-chain/node/issues/161#issuecomment-438600434
	FromHeightInclusive int64

	PublishKafka bool `mapstructure:"publishKafka"`

	// Start a local publisher which publish all topics into an auto-rotation json file
	// For full-node user and debugging usage
	PublishLocal bool `mapstructure:"publishLocal"`
	// refer: https://github.com/natefinch/lumberjack/blob/7d6a1875575e09256dc552b4c0e450dcd02bd10e/lumberjack.go#L85-L87
	LocalMaxSize int `mapstructure:"localMaxSize"`
	// refer: https://github.com/natefinch/lumberjack/blob/7d6a1875575e09256dc552b4c0e450dcd02bd10e/lumberjack.go#L89-L94
	LocalMaxAge int `mapstructure:"localMaxAge"`
}

func defaultPublicationConfig() *PublicationConfig {
	return &PublicationConfig{
		PublishOrderUpdates: false,
		OrderUpdatesTopic:   "orders",
		OrderUpdatesKafka:   "127.0.0.1:9092",

		PublishAccountBalance: false,
		AccountBalanceTopic:   "accounts",
		AccountBalanceKafka:   "127.0.0.1:9092",

		PublishOrderBook: false,
		OrderBookTopic:   "orders",
		OrderBookKafka:   "127.0.0.1:9092",

		PublishBlockFee: false,
		BlockFeeTopic:   "accounts",
		BlockFeeKafka:   "127.0.0.1:9092",

		PublishTransfer: false,
		TransferTopic:   "transfers",
		TransferKafka:   "127.0.0.1:9092",

		PublicationChannelSize: 10000,
		FromHeightInclusive:    1,
		PublishKafka:           false,
		PublishLocal:           false,
		LocalMaxSize:           1024,
		LocalMaxAge:            7,
	}
}

func (pubCfg PublicationConfig) ShouldPublishAny() bool {
	return pubCfg.PublishOrderUpdates ||
		pubCfg.PublishAccountBalance ||
		pubCfg.PublishOrderBook ||
		pubCfg.PublishBlockFee ||
		pubCfg.PublishTransfer
}

type LogConfig struct {
	LogToConsole bool   `mapstructure:"logToConsole"`
	LogFileRoot  string `mapstructure:"logFileRoot"`
	LogFilePath  string `mapstructure:"logFilePath"`
	LogBuffSize  int64  `mapstructure:"logBuffSize"`
}

func defaultLogConfig() *LogConfig {
	return &LogConfig{
		LogToConsole: true,
		LogFileRoot:  "",
		LogFilePath:  "bnc.log",
		LogBuffSize:  10000,
	}
}

type BaseConfig struct {
	AccountCacheSize          int   `mapstructure:"accountCacheSize"`
	SignatureCacheSize        int   `mapstructure:"signatureCacheSize"`
	StartMode                 uint8 `mapstructure:"startMode"`
	BreatheBlockInterval      int   `mapstructure:"breatheBlockInterval"`
	OrderKeeperConcurrency    uint  `mapstructure:"orderKeeperConcurrency"`
	BreatheBlockDaysCountBack int   `mapstructure:"breatheBlockDaysCountBack"`
}

func defaultBaseConfig() *BaseConfig {
	return &BaseConfig{
		AccountCacheSize:          30000,
		SignatureCacheSize:        30000,
		StartMode:                 0,
		BreatheBlockInterval:      0,
		OrderKeeperConcurrency:    2,
		BreatheBlockDaysCountBack: 7,
	}
}

func (context *BinanceChainContext) ParseAppConfigInPlace() error {
	// this piece of code should be consistent with bindFlagsLoadViper
	// vendor/github.com/tendermint/tendermint/libs/cli/setup.go:125
	homeDir := viper.GetString(cli.HomeFlag)
	context.Viper.SetConfigName(AppConfigFileName)
	context.Viper.AddConfigPath(homeDir)
	context.Viper.AddConfigPath(filepath.Join(homeDir, "config"))

	// If a config file is found, read it in.
	if err := context.Viper.ReadInConfig(); err == nil {
		// stderr, so if we redirect output to json file, this doesn't appear
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// ignore not found error, return other errors
		return err
	}

	err := context.Viper.Unmarshal(context.BinanceChainConfig)
	if err != nil {
		return err
	}
	return nil
}

// WriteConfigFile renders config using the template and writes it to configFilePath.
func WriteConfigFile(configFilePath string, config *BinanceChainConfig) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	common.MustWriteFile(configFilePath, buffer.Bytes(), 0644)
}
