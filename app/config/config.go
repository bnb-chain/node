package config

import (
	"bytes"
	"math"
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

[upgrade]
# Block height of BEP6 upgrade
BEP6Height = {{ .UpgradeConfig.BEP6Height }}
# Block height of BEP9 upgrade
BEP9Height = {{ .UpgradeConfig.BEP9Height }}
# Block height of BEP10 upgrade
BEP10Height = {{ .UpgradeConfig.BEP10Height }}
# Block height of BEP19 upgrade
BEP19Height = {{ .UpgradeConfig.BEP19Height }}
# Block height of BEP12 upgrade
BEP12Height = {{ .UpgradeConfig.BEP12Height }}
# Block height of BEP3 upgrade
BEP3Height = {{ .UpgradeConfig.BEP3Height }}
# Block height of FixSignBytesOverflow upgrade
FixSignBytesOverflowHeight = {{ .UpgradeConfig.FixSignBytesOverflowHeight }}
# Block height of LotSizeOptimization upgrade
LotSizeUpgradeHeight = {{ .UpgradeConfig.LotSizeUpgradeHeight }}
# Block height of changing listing rule upgrade
ListingRuleUpgradeHeight = {{ .UpgradeConfig.ListingRuleUpgradeHeight }}
# Block height of FixZeroBalanceHeight upgrade
FixZeroBalanceHeight = {{ .UpgradeConfig.FixZeroBalanceHeight }}
# Block height of smart chain upgrade
LaunchBscUpgradeHeight = {{ .UpgradeConfig.LaunchBscUpgradeHeight }}

[query]
# ABCI query interface black list, suggested value: ["custom/gov/proposals", "custom/timelock/timelocks", "custom/atomicSwap/swapcreator", "custom/atomicSwap/swaprecipient"]
ABCIQueryBlackList = {{ .QueryConfig.ABCIQueryBlackList }}

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

# Whether we want publish block
publishBlock = {{ .PublicationConfig.PublishBlock }}
blockTopic = "{{ .PublicationConfig.BlockTopic }}"
blockKafka = "{{ .PublicationConfig.BlockKafka }}"

# Whether we want publish distribution
publishDistributeReward = {{ .PublicationConfig.PublishDistributeReward }}
distributeRewardTopic = "{{ .PublicationConfig.DistributeRewardTopic }}"
distributeRewardKafka = "{{ .PublicationConfig.DistributeRewardKafka }}"

# Whether we want publish staking
publishStaking = {{ .PublicationConfig.PublishStaking }}
stakingTopic = "{{ .PublicationConfig.StakingTopic }}"
stakingKafka = "{{ .PublicationConfig.StakingKafka }}"

# Whether we want publish slashing
publishSlashing = {{ .PublicationConfig.PublishSlashing }}
slashingTopic = "{{ .PublicationConfig.SlashingTopic }}"
slashingKafka = "{{ .PublicationConfig.SlashingKafka }}"

# Whether we want publish cross transfer
publishCrossTransfer = {{ .PublicationConfig.PublishCrossTransfer }}
crossTransferTopic = "{{ .PublicationConfig.CrossTransferTopic }}"
crossTransferKafka = "{{ .PublicationConfig.CrossTransferKafka }}"

# Whether we want publish side proposals
publishSideProposal = {{ .PublicationConfig.PublishSideProposal }}
sideProposalTopic = "{{ .PublicationConfig.SideProposalTopic }}"
sideProposalKafka = "{{ .PublicationConfig.SideProposalKafka }}"

# Global setting
publicationChannelSize = {{ .PublicationConfig.PublicationChannelSize }}
publishKafka = {{ .PublicationConfig.PublishKafka }}
publishLocal = {{ .PublicationConfig.PublishLocal }}
# max size in megabytes of marketdata json file before rotate
localMaxSize = {{ .PublicationConfig.LocalMaxSize }}
# max days of marketdata json files to keep before deleted
localMaxAge = {{ .PublicationConfig.LocalMaxAge }}

# whether the kafka open SASL_PLAINTEXT auth
auth = {{ .PublicationConfig.Auth }}
kafkaUserName = "{{ .PublicationConfig.KafkaUserName }}"
kafkaPassword = "{{ .PublicationConfig.KafkaPassword }}"

# stop process when publish to Kafka failed
stopOnKafkaFail = {{ .PublicationConfig.StopOnKafkaFail }}

# please modify the default value into the version of Kafka you are using
# kafka broker version, default (and most recommended) is 2.1.0. Minimal supported version could be 0.8.2.0
kafkaVersion = "{{ .PublicationConfig.KafkaVersion }}"

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

[cross_chain]
# IBC chain-id for current chain
ibcChainId = {{ .CrossChainConfig.IbcChainId }}
# chain-id for bsc chain
bscChainId = "{{ .CrossChainConfig.BscChainId }}"
# IBC chain-id for bsc chain
bscIbcChainId = {{ .CrossChainConfig.BscIbcChainId }}

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
	*UpgradeConfig     `mapstructure:"upgrade"`
	*QueryConfig       `mapstructure:"query"`
	*CrossChainConfig  `mapstructure:"cross_chain"`
}

func DefaultBinanceChainConfig() *BinanceChainConfig {
	return &BinanceChainConfig{
		AddressConfig:     defaultAddressConfig(),
		PublicationConfig: defaultPublicationConfig(),
		LogConfig:         defaultLogConfig(),
		BaseConfig:        defaultBaseConfig(),
		UpgradeConfig:     defaultUpgradeConfig(),
		QueryConfig:       defaultQueryConfig(),
		CrossChainConfig:  defaultCrossChainConfig(),
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

	PublishBlock bool   `mapstructure:"publishBlock"`
	BlockTopic   string `mapstructure:"blockTopic"`
	BlockKafka   string `mapstructure:"blockKafka"`

	PublishDistributeReward bool   `mapstructure:"publishDistributeReward"`
	DistributeRewardTopic   string `mapstructure:"distributeRewardTopic"`
	DistributeRewardKafka   string `mapstructure:"distributeRewardKafka"`

	PublishStaking bool   `mapstructure:"publishStaking"`
	StakingTopic   string `mapstructure:"stakingTopic"`
	StakingKafka   string `mapstructure:"stakingKafka"`

	PublishSlashing bool   `mapstructure:"publishSlashing"`
	SlashingTopic   string `mapstructure:"slashingTopic"`
	SlashingKafka   string `mapstructure:"slashingKafka"`

	PublishCrossTransfer bool   `mapstructure:"publishCrossTransfer"`
	CrossTransferTopic   string `mapstructure:"crossTransferTopic"`
	CrossTransferKafka   string `mapstructure:"crossTransferKafka"`

	PublishSideProposal bool   `mapstructure:"publishSideProposal"`
	SideProposalTopic   string `mapstructure:"sideProposalTopic"`
	SideProposalKafka   string `mapstructure:"sideProposalKafka"`

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

	Auth            bool   `mapstructure:"auth"`
	StopOnKafkaFail bool   `mapstructure:"stopOnKafkaFail"`
	KafkaUserName   string `mapstructure:"kafkaUserName"`
	KafkaPassword   string `mapstructure:"kafkaPassword"`

	KafkaVersion string `mapstructure:"kafkaVersion"`
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

		PublishBlock: false,
		BlockTopic:   "block",
		BlockKafka:   "127.0.0.1:9092",

		PublishDistributeReward: false,
		DistributeRewardTopic:   "distribution",
		DistributeRewardKafka:   "127.0.0.1:9092",

		PublishStaking: false,
		StakingTopic:   "staking",
		StakingKafka:   "127.0.0.1:9092",

		PublishSlashing: false,
		SlashingTopic:   "slashing",
		SlashingKafka:   "127.0.0.1:9092",

		PublishCrossTransfer: false,
		CrossTransferTopic:   "crossTransfer",
		CrossTransferKafka:   "127.0.0.1:9092",

		PublishSideProposal: false,
		SideProposalTopic:   "sideProposal",
		SideProposalKafka:   "127.0.0.1:9092",

		PublicationChannelSize: 10000,
		FromHeightInclusive:    1,
		PublishKafka:           false,

		PublishLocal: false,
		LocalMaxSize: 1024,
		LocalMaxAge:  7,

		Auth:            false,
		KafkaUserName:   "",
		KafkaPassword:   "",
		StopOnKafkaFail: false,

		KafkaVersion: "2.1.0",
	}
}

func (pubCfg PublicationConfig) ShouldPublishAny() bool {
	return pubCfg.PublishOrderUpdates ||
		pubCfg.PublishAccountBalance ||
		pubCfg.PublishOrderBook ||
		pubCfg.PublishBlockFee ||
		pubCfg.PublishTransfer ||
		pubCfg.PublishBlock ||
		pubCfg.PublishDistributeReward ||
		pubCfg.PublishStaking ||
		pubCfg.PublishSlashing ||
		pubCfg.PublishCrossTransfer ||
		pubCfg.PublishSideProposal
}

type CrossChainConfig struct {
	IbcChainId uint16 `mapstructure:"ibcChainId"`

	BscChainId    string `mapstructure:"bscChainId"`
	BscIbcChainId uint16 `mapstructure:"bscIBCChainId"`
}

func defaultCrossChainConfig() *CrossChainConfig {
	return &CrossChainConfig{
		IbcChainId: 1,

		BscChainId:    "bsc",
		BscIbcChainId: 2,
	}
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

type UpgradeConfig struct {

	// Galileo Upgrade
	BEP6Height  int64 `mapstructure:"BEP6Height"`
	BEP9Height  int64 `mapstructure:"BEP9Height"`
	BEP10Height int64 `mapstructure:"BEP10Height"`
	BEP19Height int64 `mapstructure:"BEP19Height"`
	// Hubble Upgrade
	BEP12Height int64 `mapstructure:"BEP12Height"`
	// Archimedes Upgrade
	BEP3Height int64 `mapstructure:"BEP3Height"`
	// Heisenberg Upgrade
	FixSignBytesOverflowHeight int64 `mapstructure:"FixSignBytesOverflowHeight"`
	LotSizeUpgradeHeight       int64 `mapstructure:"LotSizeUpgradeHeight"`
	ListingRuleUpgradeHeight   int64 `mapstructure:"ListingRuleUpgradeHeight"`
	FixZeroBalanceHeight       int64 `mapstructure:"FixZeroBalanceHeight"`
	// TODO: add upgrade name
	LaunchBscUpgradeHeight int64 `mapstructure:"LaunchBscUpgradeHeight"`
}

func defaultUpgradeConfig() *UpgradeConfig {
	// make the upgraded functions enabled by default
	return &UpgradeConfig{
		BEP6Height:                 1,
		BEP9Height:                 1,
		BEP10Height:                1,
		BEP19Height:                1,
		BEP12Height:                1,
		BEP3Height:                 1,
		FixSignBytesOverflowHeight: 1,
		LotSizeUpgradeHeight:       1,
		ListingRuleUpgradeHeight:   1,
		FixZeroBalanceHeight:       1,
		LaunchBscUpgradeHeight:     math.MaxInt64,
	}
}

type QueryConfig struct {
	ABCIQueryBlackList []string `mapstructure:"ABCIQueryBlackList"`
}

func defaultQueryConfig() *QueryConfig {
	return &QueryConfig{
		ABCIQueryBlackList: nil,
	}
}

type DexConfig struct {
	BUSDSymbol string `mapstructure:"BUSDSymbol"`
}

func defaultGovConfig() *DexConfig {
	return &DexConfig{
		BUSDSymbol: "",
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
