package sub

import (
	"github.com/cosmos/cosmos-sdk/pubsub"

	"github.com/binance-chain/node/plugins/bridge"
)

func SubscribeCrossTransferEvent(sub *pubsub.Subscriber) error {
	err := sub.Subscribe(bridge.Topic, func(event pubsub.Event) {
		switch event.(type) {
		case bridge.CrossTransferEvent:
			crossTransferEvent := event.(bridge.CrossTransferEvent)
			if stagingArea.CrossTransferData == nil {
				stagingArea.CrossTransferData = make([]bridge.CrossTransferEvent, 0, 1)
			}
			stagingArea.CrossTransferData = append(stagingArea.CrossTransferData, crossTransferEvent)

		default:
			sub.Logger.Info("unknown event type")
		}
	})
	return err
}

func commitCrossTransfer() {
	if len(stagingArea.CrossTransferData) > 0 {
		toPublish.EventData.CrossTransferData = append(toPublish.EventData.CrossTransferData, stagingArea.CrossTransferData...)
	}
}
