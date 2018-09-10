package config

import (
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/viper"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"os"
)

type BinanceChainContext struct {
	Config *BinanceChainConfig
	Logger log.Logger
}

func NewDefaultContext() *BinanceChainContext {
	return &BinanceChainContext{DefaultBinanceChainConfig(), log.NewTMLogger(log.NewSyncWriter(os.Stdout))}
}

func (context *BinanceChainContext) ToCosmosServerCtx() *server.Context {
	return &server.Context{Config: &context.Config.Config, Logger: context.Logger}
}

type BinanceChainConfig struct {
	tmcfg.Config `mapstructure:",squash"`

	// Extended otions for Binance Chain
	Publication *PublicationConfig `mapstructure:"publication"`
}

type PublicationConfig struct {
	PublishMarketData bool   `mapstructure:"publishMarketData"`
	MarketDataTopic   string `mapstructure:"marketDataTopic"`
	MarketDataKafka   string `mapstructure:"marketDataKafka"`

	PublishAccountBalance bool   `mapstructure:"publishAccountBalance"`
	AccountBalanceTopic   string `mapstructure:"accountBalanceTopic"`
	AccountBalanceKafka   string `mapstructure:"accountBalanceKafka"`

	PublishOrderBook bool   `mapstructure:"publishOrderBook"`
	OrderBookTopic   string `mapstructure:"orderBookTopic"`
	OrderBookKafka   string `mapstructure:"orderBookKafka"`
}

func DefaultBinanceChainConfig() *BinanceChainConfig {
	return &BinanceChainConfig{
		Config:      *tmcfg.DefaultConfig(),
		Publication: DefaultPublicationConfig(),
	}
}

func DefaultPublicationConfig() *PublicationConfig {
	return &PublicationConfig{
		PublishMarketData:     false,
		PublishAccountBalance: false,
		PublishOrderBook:      false,
		MarketDataTopic:       "test",
		AccountBalanceTopic:   "accounts",
		OrderBookTopic:        "books",
	}
}

func (context *BinanceChainContext) ParseConfig() (*BinanceChainConfig, error) {
	err := viper.Unmarshal(context.Config)
	if err != nil {
		return nil, err
	}
	context.Config.SetRoot(context.Config.RootDir)
	tmcfg.EnsureRoot(context.Config.RootDir)
	return context.Config, err
}
