package p2p

import (
	"sync"

	"github.com/binance-chain/tss-lib/tss"

	"github.com/binance-chain/tss/common"
)

var once = sync.Once{}
var registeredTransporters map[common.TssClientId]*memTransporter

// in memory transporter used for testing
type memTransporter struct {
	cid       common.TssClientId
	receiveCh chan common.P2pMessageWrapper
}

var _ common.Transporter = (*memTransporter)(nil)

func NewMemTransporter(cid common.TssClientId) common.Transporter {
	t := memTransporter{
		cid:       cid,
		receiveCh: make(chan common.P2pMessageWrapper, receiveChBufSize),
	}
	once.Do(func() {
		registeredTransporters = make(map[common.TssClientId]*memTransporter, 0)
	})

	registeredTransporters[cid] = &t
	return &t
}

func GetMemTransporter(cid common.TssClientId) common.Transporter {
	return registeredTransporters[cid]
}

func (t *memTransporter) NodeKey() []byte {
	return []byte(t.cid.String())
}

func (t *memTransporter) Broadcast(msg tss.Message) error {
	logger.Debugf("[%s] Broadcast: %s", t.cid, msg)
	for cid, peer := range registeredTransporters {
		if cid != t.cid {
			originMsg, _, _ := msg.WireBytes()
			peer.receiveCh <- common.P2pMessageWrapper{MessageWrapperBytes: originMsg}
		}
	}
	return nil
}

func (t *memTransporter) Send(msg []byte, to common.TssClientId) error {
	logger.Debugf("[%s] Sending: %x", t.cid, msg)
	if peer, ok := registeredTransporters[to]; ok {
		peer.receiveCh <- common.P2pMessageWrapper{MessageWrapperBytes: msg}
	}
	return nil
}

func (t *memTransporter) ReceiveCh() <-chan common.P2pMessageWrapper {
	return t.receiveCh
}

func (t *memTransporter) Shutdown() error {
	return nil
}
