package sub

import (
	"time"

	"github.com/binance-chain/node/plugins/bridge"
	"github.com/cosmos/cosmos-sdk/pubsub"
)

func SubscribeAllEvent(sub *pubsub.Subscriber) error {
	if err := SubscribeStakeEvent(sub); err != nil {
		return err
	}
	if err := SubscribeSlashEvent(sub); err != nil {
		return err
	}
	if err := SubscribeCrossTransferEvent(sub); err != nil {
		return err
	}

	// commit events data from staging area to 'toPublish' when receiving `TxDeliverEvent`, represents the tx is successfully delivered.
	if err := sub.Subscribe(TxDeliverTopic, func(event pubsub.Event) {
		switch event.(type) {
		case TxDeliverSuccEvent:
			commit()
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
}

func newEventStore() *EventStore {
	return &EventStore{
		StakeData: &StakeData{},
	}
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

func commit() {
	commitStake()
	commitCrossTransfer()
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
