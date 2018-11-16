package config

import (
	"bytes"
	"path/filepath"
	"text/template"

	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/server"

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

[log]

# Write logs to console instead of file
logToConsole = {{ .LogConfig.LogToConsole }}

## The below parameters take effect only when logToConsole is false
# Log file path relative to home path
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
}

func DefaultBinanceChainConfig() *BinanceChainConfig {
	return &BinanceChainConfig{
		AddressConfig:     defaultAddressConfig(),
		PublicationConfig: defaultPublicationConfig(),
		LogConfig:         defaultLogConfig(),
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
		Bech32PrefixAccAddr:  "bnc",
		Bech32PrefixAccPub:   "bncp",
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
	}
}

func (pubCfg PublicationConfig) ShouldPublishAny() bool {
	return pubCfg.PublishOrderUpdates || pubCfg.PublishAccountBalance || pubCfg.PublishOrderBook
}

type LogConfig struct {
	LogToConsole bool   `mapstructure:"logToConsole"`
	LogFilePath  string `mapstructure:"logFilePath"`
	LogBuffSize  int64  `mapstructure:"logBuffSize"`
}

func defaultLogConfig() *LogConfig {
	return &LogConfig{
		LogToConsole: true,
		LogFilePath:  "bnc.log",
		LogBuffSize:  10000,
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
