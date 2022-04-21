package types

import (
	"github.com/cosmos/cosmos-sdk/pubsub"
)

const (
	Topic = pubsub.Topic("oracle-event")
)

type CrossAppFailEvent struct {
	TxHash     string
	ChainId    string
	RelayerFee int64
	From       string
}

func (event CrossAppFailEvent) GetTopic() pubsub.Topic {
	return Topic
}
