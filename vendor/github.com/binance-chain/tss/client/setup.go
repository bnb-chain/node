package client

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

func Setup(cfg common.TssConfig) {
	err := os.Mkdir("./configs", 0700)
	if err != nil {
		common.Panic(err)
	}
	allPeerIds := make([]string, 0, cfg.Parties)
	for i := 0; i < cfg.Parties; i++ {
		configPath := fmt.Sprintf("./configs/%d", i)
		err := os.Mkdir(configPath, 0700)
		if err != nil {
			common.Panic(err)
		}
		// generate node identifier key
		privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			common.Panic(err)
		}

		pid, err := peer.IDFromPublicKey(privKey.GetPublic())
		if err != nil {
			common.Panic(err)
		}
		allPeerIds = append(allPeerIds, fmt.Sprintf("%s@%s", fmt.Sprintf("party%d", i), pid.Pretty()))

		bytes, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			common.Panic(err)
		}
		ioutil.WriteFile(configPath+"/node_key", bytes, os.FileMode(0600))
	}

	for i := 0; i < cfg.Parties; i++ {
		configFilePath := fmt.Sprintf("./configs/%d/config.json", i)
		tssConfig := cfg
		tssConfig.P2PConfig.ExpectedPeers = make([]string, cfg.Parties, cfg.Parties)
		copy(tssConfig.P2PConfig.ExpectedPeers, allPeerIds)
		tssConfig.P2PConfig.ExpectedPeers = append(tssConfig.P2PConfig.ExpectedPeers[:i], tssConfig.P2PConfig.ExpectedPeers[i+1:]...)

		if cfg.Parties == len(cfg.P2PConfig.PeerAddrs) {
			tssConfig.P2PConfig.PeerAddrs = make([]string, cfg.Parties, cfg.Parties)
			copy(tssConfig.P2PConfig.PeerAddrs, cfg.P2PConfig.PeerAddrs)
			tssConfig.P2PConfig.PeerAddrs = append(tssConfig.P2PConfig.PeerAddrs[:i], tssConfig.P2PConfig.PeerAddrs[i+1:]...)
		}

		tssConfig.Id = p2p.GetClientIdFromExpectedPeers(allPeerIds[i])
		tssConfig.Moniker = p2p.GetMonikerFromExpectedPeers(allPeerIds[i])

		bytes, err := json.MarshalIndent(&tssConfig, "", "    ")
		if err != nil {
			common.Panic(err)
		}
		ioutil.WriteFile(configFilePath, bytes, os.FileMode(0600))
	}
}
