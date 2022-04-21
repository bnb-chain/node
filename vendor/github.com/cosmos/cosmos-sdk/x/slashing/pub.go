package slashing

import (
	"time"

	"github.com/cosmos/cosmos-sdk/pubsub"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const Topic = pubsub.Topic("slashing")

type SideSlashEvent struct {
	Validator              sdk.ValAddress
	InfractionType         byte
	InfractionHeight       int64
	SlashHeight            int64
	JailUtil               time.Time
	SlashAmt               int64
	ToFeePool              int64
	SideChainId            string
	Submitter              sdk.AccAddress
	SubmitterReward        int64
	ValidatorsCompensation map[string]int64
}

func (event SideSlashEvent) GetTopic() pubsub.Topic {
	return Topic
}
