package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/cmd/pressuremaker/utils"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

type PressureMakerConfig struct {
	config.PublicationConfig `mapstructure:"publication"`

	NumOfTradesPerBlock   int    `mapstructure:"numOfTradesPerBlock"`
	NumOfTransferPerBlock int    `mapstructure:"numOfTransferPerBlock"`
	Blocks                int    `mapstructure:"numOfBlocks"`
	BlockIntervalMs       int    `mapstructure:"blockIntervalMs"`
	PressureMode          int    `mapstructure:"mode"`
	PrometheusAddr        string `mapstructure:"prometheusAddr"`
}

func main() {
	Execute()
}

var cfg = PressureMakerConfig{}
var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/config.toml)")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("config")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
	viper.Unmarshal(&cfg)
}

var rootCmd = &cobra.Command{
	Use:   "kafka_pressure_maker",
	Short: "generate legal trade and order and accounts messages into kafka",
	Run: func(cmd *cobra.Command, args []string) {
		context := server.NewDefaultContext() // used to get logger

		// TODO: find an elegant way to exit
		// The problem of shutdown is publication is async (we don't know when messages are
		finishSignal := make(chan struct{})
		publisher := pub.NewKafkaMarketDataPublisher(context.Logger)

		srv := &http.Server{
			Addr: cfg.PrometheusAddr,
			Handler: promhttp.InstrumentMetricHandler(
				prometheus.DefaultRegisterer, promhttp.HandlerFor(
					prometheus.DefaultGatherer,
					promhttp.HandlerOpts{MaxRequestsInFlight: 10},
				),
			),
		}
		go func() {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				// Error starting or closing listener:
				fmt.Printf("Prometheus HTTP server ListenAndServe, err=%v\n", err)
			}
		}()

		generator := utils.MessageGenerator{
			NumOfTradesPerBlock:   cfg.NumOfTradesPerBlock,
			NumOfTransferPerBlock: cfg.NumOfTransferPerBlock,
			NumOfBlocks:           cfg.Blocks,
			OrderChangeMap:        make(orderPkg.OrderInfoForPublish, 0),
			TimeStart:             time.Now(),
		}
		generator.Setup()

		for h := 1; h <= generator.NumOfBlocks; h++ {
			var tradesToPublish []*pub.Trade
			var orderChanges orderPkg.OrderChanges
			var accounts map[string]pub.Account
			var transfers *pub.Transfers
			timeNow := generator.TimeStart.Add(time.Second * time.Duration(h))
			timePub := timeNow.Unix()
			switch cfg.PressureMode {
			case 1:
				// each trade has two equal quantity order
				tradesToPublish, orderChanges, accounts, transfers = generator.OneOnOneMessages(h, timeNow)
			case 2:
				// each big order eat two small orders
				tradesToPublish, orderChanges, accounts, transfers = generator.TwoOnOneMessages(h, timeNow)
			case 3:
				// simulate 1 million expire orders to publish at breathe block
				tradesToPublish, orderChanges, accounts = generator.ExpireMessages(h, timeNow)
			}
			time.Sleep(time.Duration(cfg.BlockIntervalMs) * time.Millisecond)
			generator.Publish(int64(h), timePub, tradesToPublish, orderChanges, generator.OrderChangeMap, accounts, transfers)
		}

		<-finishSignal
		publisher.Stop()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
