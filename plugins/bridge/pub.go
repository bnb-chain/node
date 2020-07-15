package bridge

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/pubsub"
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/bridge/keeper"
)

const (
	Topic = pubsub.Topic("cross-transfer")

	TransferOutType string = "TO"
	TransferInType  string = "TI"
	TransferAckRefundType string = "TAR"
	TransferFailAckRefundType string = "TFAR"

	TransferBindType string = "TB"
	TransferUnBindType string = "TUB"
	TransferFailBindType string = "TFB"
	TransferApproveBindType string = "TPB"

)

type CrossTransferEvent struct {
	TxHash  string
	ChainId string
	Type    string
	RelayerFee int64
	From    string
	Denom   string
	To      []CrossReceiver
}

type CrossReceiver struct {
	Addr   string
	Amount int64
}

func (event CrossTransferEvent) GetTopic() pubsub.Topic {
	return Topic
}

func publishCrossChainEvent(ctx types.Context, keeper keeper.Keeper, from string, to []CrossReceiver, symbol string, eventType string, relayerFee int64) {
	if keeper.PbsbServer != nil {
		txHash := ctx.Value(baseapp.TxHashKey)
		if txHashStr, ok := txHash.(string); ok {
			event := CrossTransferEvent{
				TxHash:  txHashStr,
				ChainId: keeper.DestChainName,
				RelayerFee: relayerFee,
				Type:    eventType,
				From:    from,
				Denom:   symbol,
				To:      to,
			}
			keeper.PbsbServer.Publish(event)
		} else {
			ctx.Logger().With("module", "bridge").Error("failed to get txhash, will not publish cross transfer event ")
		}
	}
}
