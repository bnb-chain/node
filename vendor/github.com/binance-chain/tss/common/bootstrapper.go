package common

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"github.com/bgentry/speakeasy"
)

type BootstrapMode uint8

const (
	KeygenMode BootstrapMode = iota
	SignMode
	PreRegroupMode
	RegroupMode
)

// Bootstrapper is helper of pre setting of each kind of client command
// Before keygen, it helps setup peers' moniker and libp2p id, in a "raw" tcp communication way
// For sign, it helps setup signers in libp2p network
// For preregroup, it helps setup new initialized peers' moniker and libp2p id, in a "raw" tcp communication way
// For regroup, it helps setup peers' old and new committee information
type Bootstrapper struct {
	ChannelId       string
	ChannelPassword string
	ExpectedPeers   int
	Msg             *BootstrapMessage
	Cfg             *TssConfig

	Peers sync.Map // id -> peerInfo
}

func NewBootstrapper(expectedPeers int, config *TssConfig) *Bootstrapper {
	// when invoke from anther process (bnbcli), we need set channel id and password here
	if config.ChannelId == "" {
		reader := bufio.NewReader(os.Stdin)
		channelId, err := GetString("please set channel id of this session", reader)
		if err != nil {
			Panic(err)
		}
		if len(channelId) != 11 {
			Panic(fmt.Errorf("channelId format is invalid"))
		}
		config.ChannelId = channelId
	}
	if config.ChannelPassword == "" {
		if p, err := speakeasy.Ask("> please input password (AGREED offline with peers) of this session:"); err == nil {
			if p == "" {
				Panic(fmt.Errorf("channel password should not be empty"))
			}
			config.ChannelPassword = p
		} else {
			Panic(err)
		}
	}

	bootstrapMsg, err := NewBootstrapMessage(
		config.ChannelId,
		config.ChannelPassword,
		config.ListenAddr,
		PeerParam{
			ChannelId: config.ChannelId,
			Moniker:   config.Moniker,
			Msg:       config.Message,
			Id:        string(config.Id),
			N:         config.Parties,
			T:         config.Threshold,
			NewN:      config.NewParties,
			NewT:      config.NewThreshold,
			IsOld:     config.IsOldCommittee,
			IsNew:     !config.IsOldCommittee,
		},
	)
	if err != nil {
		Panic(err)
	}

	return &Bootstrapper{
		ChannelId:       config.ChannelId,
		ChannelPassword: config.ChannelPassword,
		ExpectedPeers:   expectedPeers,
		Msg:             bootstrapMsg,
		Cfg:             config,
	}
}

func (b *Bootstrapper) HandleBootstrapMsg(peerMsg BootstrapMessage) error {
	if peerParam, err := Decrypt(peerMsg.PeerInfo, b.ChannelId, b.ChannelPassword); err != nil {
		return err
	} else {
		if info, ok := b.Peers.Load(peerParam.Id); info != nil && ok {
			if peerParam.Moniker != info.(PeerInfo).Moniker {
				return fmt.Errorf("received different moniker for id: %s", peerParam.Id)
			}
		} else {
			if peerParam.N != TssCfg.Parties {
				return fmt.Errorf("received differetnt n for party: %s, %s", peerParam.Moniker, peerParam.Id)
			}
			if peerParam.T != TssCfg.Threshold {
				return fmt.Errorf("received different t for party: %s, %s", peerParam.Moniker, peerParam.Id)
			}
			if peerParam.Msg != TssCfg.Message {
				return fmt.Errorf("received different message to be signed for party: %s, %s", peerParam.Moniker, peerParam.Id)
			}
			if peerParam.NewN != TssCfg.NewParties {
				return fmt.Errorf("received different new n for party: %s, %s", peerParam.Moniker, peerParam.Id)
			}
			if peerParam.NewT != TssCfg.NewThreshold {
				return fmt.Errorf("received different new t for party: %s, %s", peerParam.Moniker, peerParam.Id)
			}

			pi := PeerInfo{
				Id:         peerParam.Id,
				Moniker:    peerParam.Moniker,
				RemoteAddr: peerMsg.Addr,
				IsOld:      peerParam.IsOld,
				IsNew:      peerParam.IsNew,
			}
			b.Peers.Store(peerParam.Id, pi)
		}
	}
	return nil
}

func (b *Bootstrapper) IsFinished() bool {
	received := b.LenOfPeers()
	switch b.Cfg.BMode {
	case KeygenMode:
		logger.Debugf("received peers: %d, expect peers: %v", received, b.ExpectedPeers)
		return received == b.ExpectedPeers
	case SignMode:
		logger.Debugf("received peers: %d, expect peers: %d", received, b.Cfg.Threshold)
		return received == b.Cfg.Threshold
	case PreRegroupMode:
		logger.Debugf("received peers: %d, expect peers: %v", received, b.Cfg.ExpectedPeers)
		return received == b.ExpectedPeers
	case RegroupMode:
		numOfOld := 0
		numOfNew := 0
		b.Peers.Range(func(_, value interface{}) bool {
			if pi, ok := value.(PeerInfo); ok {
				if pi.IsOld {
					numOfOld++
				}
				if pi.IsNew {
					numOfNew++
				}
			}
			return true
		})
		if TssCfg.IsOldCommittee && TssCfg.IsNewCommittee {
			return numOfOld >= b.Cfg.Threshold && numOfNew+1 >= b.Cfg.NewParties
		} else if TssCfg.IsOldCommittee && !TssCfg.IsNewCommittee {
			return numOfOld >= b.Cfg.Threshold && numOfNew >= b.Cfg.NewParties
		} else if !TssCfg.IsOldCommittee && TssCfg.IsNewCommittee {
			return numOfOld >= b.Cfg.Threshold+1 && numOfNew+1 >= b.Cfg.NewParties
		} else {
			return numOfOld >= b.Cfg.Threshold+1 && numOfNew >= b.Cfg.NewParties
		}
	default:
		return false
	}
}

func (b *Bootstrapper) LenOfPeers() int {
	received := 0
	b.Peers.Range(func(_, _ interface{}) bool {
		received++
		return true
	})
	return received
}

type PeerInfo struct {
	Id         string
	Moniker    string
	RemoteAddr string
	IsOld      bool
	IsNew      bool
}
