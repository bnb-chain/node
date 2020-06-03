package sub

import (
	"time"

	"github.com/cosmos/cosmos-sdk/pubsub"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
)

type SlashData struct {
	Validator        sdk.ValAddress
	InfractionType   byte
	InfractionHeight int64
	JailUtil         time.Time
	SlashAmount      int64
	Submitter        sdk.AccAddress
	SubmitterReward  int64
}

func SubscribeSlashEvent(sub *pubsub.Subscriber) error {
	err := sub.Subscribe(slashing.Topic, func(event pubsub.Event) {
		switch event.(type) {
		case slashing.SideSlashEvent:
			sideSlashEvent := event.(slashing.SideSlashEvent)
			if toPublish.EventData.SlashData == nil {
				toPublish.EventData.SlashData = make(map[string][]SlashData)
			}
			if _, ok := toPublish.EventData.SlashData[sideSlashEvent.SideChainId]; !ok {
				toPublish.EventData.SlashData[sideSlashEvent.SideChainId] = make([]SlashData, 0)
			}
			toPublish.EventData.SlashData[sideSlashEvent.SideChainId] = append(toPublish.EventData.SlashData[sideSlashEvent.SideChainId], SlashData{
				Validator:        sideSlashEvent.Validator,
				InfractionType:   sideSlashEvent.InfractionType,
				InfractionHeight: sideSlashEvent.InfractionHeight,
				JailUtil:         sideSlashEvent.JailUtil,
				SlashAmount:      sideSlashEvent.SubmitterReward,
				Submitter:        sideSlashEvent.Submitter,
				SubmitterReward:  sideSlashEvent.SubmitterReward,
			})
		default:
			sub.Logger.Info("unknown event type")
		}
	})
	return err
}
