package sub

import (
	"time"

	"github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/plugins/bridge"

	"github.com/cosmos/cosmos-sdk/pubsub"
)

func SubscribeEvent(sub *pubsub.Subscriber, cfg *config.PublicationConfig) error {
	if cfg.PublishStaking {
		if err := SubscribeStakeEvent(sub); err != nil {
			return err
		}
	}

	if cfg.PublishSlashing {
		if err := SubscribeSlashEvent(sub); err != nil {
			return err
		}
	}

	if cfg.PublishCrossTransfer {
		if err := SubscribeCrossTransferEvent(sub); err != nil {
			return err
		}
		if err := SubscribeOracleEvent(sub); err != nil {
			return err
		}
	}

	if cfg.PublishMirror {
		if err := SubscribeMirrorEvent(sub); err != nil {
			return err
		}
	}

	// commit events data from staging area to 'toPublish' when receiving `TxDeliverEvent`, represents the tx is successfully delivered.
	if err := sub.Subscribe(TxDeliverTopic, func(event pubsub.Event) {
		switch event.(type) {
		case TxDeliverSuccEvent:
			commit(cfg)
		case TxDeliverFailEvent:
			discard()
		default:
			sub.Logger.Debug("unknown event")
		}
	}); err != nil {
		return err
	}

	return nil
}

//-----------------------------------------------------
var (
	// events to be published, should be cleaned up each block
	toPublish = &ToPublishEvent{EventData: newEventStore()}
	// staging area for accepting events to store
	// should be moved to 'toPublish' when related tx successfully delivered
	stagingArea = &EventStore{}
)

type ToPublishEvent struct {
	Height         int64
	Timestamp      time.Time
	IsBreatheBlock bool
	EventData      *EventStore
}

type EventStore struct {
	// store for stake topic
	StakeData *StakeData
	// store for slash topic
	SlashData map[string][]SlashData
	// store for cross chain transfer topic
	CrossTransferData []bridge.CrossTransferEvent
	// store for mirror topic
	MirrorData []bridge.MirrorEvent
}

func newEventStore() *EventStore {
	return &EventStore{}
}

func Clear() {
	toPublish = &ToPublishEvent{EventData: newEventStore()}
	stagingArea = newEventStore()
}

func ToPublish() *ToPublishEvent {
	return toPublish
}

func SetMeta(height int64, timestamp time.Time, isBreatheBlock bool) {
	toPublish.Height = height
	toPublish.Timestamp = timestamp
	toPublish.IsBreatheBlock = isBreatheBlock
}

func commit(cfg *config.PublicationConfig) {
	if cfg.PublishStaking {
		commitStake()
	}
	if cfg.PublishCrossTransfer {
		commitCrossTransfer()
	}
	if cfg.PublishMirror {
		commitMirror()
	}
	// clear stagingArea data
	stagingArea = newEventStore()
}

func discard() {
	stagingArea = newEventStore()
}

//---------------------------------------------------------------------
const TxDeliverTopic = pubsub.Topic("TxDeliver")

type TxDeliverEvent struct{}

func (event TxDeliverEvent) GetTopic() pubsub.Topic {
	return TxDeliverTopic
}

type TxDeliverSuccEvent struct {
	TxDeliverEvent
}
type TxDeliverFailEvent struct {
	TxDeliverEvent
}
