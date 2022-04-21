package crypto

import (
	_ "github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
	"github.com/ipfs/go-log"
	"github.com/spf13/viper"

	tmcrypto "github.com/tendermint/tendermint/crypto"
)

func NewPrivKeyTss(home, vault, passphrase, message string) (tmcrypto.PrivKey, error) {
	err := common.ReadConfigFromHome(viper.New(), false, home, vault, passphrase)
	if err != nil {
		return nil, err
	}
	common.TssCfg.Home = home
	common.TssCfg.Vault = vault
	common.TssCfg.Password = passphrase
	// the message passed here is only used to make sure peer's message are same with us, actual message to be signed should be passed through sign method
	// TODO: make this more elegant
	common.TssCfg.Message = message
	initLogLevel(common.TssCfg)
	return client.NewTssClient(&common.TssCfg, client.SignMode, false), nil
}

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.LogLevel)
	log.SetLogLevel("tss-lib", cfg.LogLevel)
	log.SetLogLevel("srv", cfg.LogLevel)
	log.SetLogLevel("trans", cfg.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
}
