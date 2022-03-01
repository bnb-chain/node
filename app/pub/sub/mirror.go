package sub

import (
	"github.com/cosmos/cosmos-sdk/pubsub"

	"github.com/bnb-chain/node/plugins/bridge"
)

func SubscribeMirrorEvent(sub *pubsub.Subscriber) error {
	err := sub.Subscribe(bridge.MirrorTopic, func(event pubsub.Event) {
		switch event.(type) {
		case bridge.MirrorEvent:
			mirrorEvent := event.(bridge.MirrorEvent)
			if stagingArea.MirrorData == nil {
				stagingArea.MirrorData = make([]bridge.MirrorEvent, 0, 1)
			}
			stagingArea.MirrorData = append(stagingArea.MirrorData, mirrorEvent)
		default:
			sub.Logger.Info("unknown event type")
		}
	})
	return err
}

func commitMirror() {
	if len(stagingArea.MirrorData) > 0 {
		toPublish.EventData.MirrorData = append(toPublish.EventData.MirrorData, stagingArea.MirrorData...)
	}
}
