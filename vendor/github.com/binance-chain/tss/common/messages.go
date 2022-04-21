package common

import (
	"encoding/gob"

	"github.com/binance-chain/tss-lib/tss"
)

func init() {
	gob.Register(DummyMsg{})
}

type DummyMsg struct {
	Content string
}

func (m DummyMsg) String() string {
	return m.Content
}

// always broadcast
func (m DummyMsg) GetTo() *tss.PartyID {
	return nil
}

func (m DummyMsg) GetFrom() *tss.PartyID {
	return nil
}

func (m DummyMsg) GetType() string {
	return ""
}

func (m DummyMsg) ValidateBasic() bool {
	return true
}

type PeerParam struct {
	ChannelId, Moniker, Msg, Id string
	N, T, NewN, NewT            int
	IsOld, IsNew                bool
}

func NewBootstrapMessage(channelId, channelPassword, addr string, param PeerParam) (*BootstrapMessage, error) {
	pi, err := Encrypt(channelPassword, param)
	if err != nil {
		return nil, err
	}
	return &BootstrapMessage{
		ChannelId: channelId,
		PeerInfo:  pi,
		Addr:      addr,
	}, nil
}
