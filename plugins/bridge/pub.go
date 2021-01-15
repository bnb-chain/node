package bridge

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/pubsub"
	"github.com/cosmos/cosmos-sdk/types"

	btype "github.com/binance-chain/node/plugins/bridge/types"

	"github.com/binance-chain/node/plugins/bridge/keeper"
)

const (
	CrossTransferTopic = pubsub.Topic("cross-transfer")

	TransferOutType           string = "TO"
	TransferInType            string = "TI"
	TransferAckRefundType     string = "TAR"
	TransferFailAckRefundType string = "TFAR"

	TransferBindType        string = "TB"
	TransferUnBindType      string = "TUB"
	TransferFailBindType    string = "TFB"
	TransferApproveBindType string = "TPB"

	CrossAppFailedType string = "CF"

	MirrorTopic           = pubsub.Topic("mirror")
	MirrorType     string = "MI"
	MirrorSyncType string = "MISY"
)

type CrossTransferEvent struct {
	TxHash     string
	ChainId    string
	Type       string
	RelayerFee int64
	From       string
	Denom      string
	Contract   string
	Decimals   int
	To         []CrossReceiver
}

type CrossReceiver struct {
	Addr   string
	Amount int64
}

func (event CrossTransferEvent) GetTopic() pubsub.Topic {
	return CrossTransferTopic
}

func publishCrossChainEvent(ctx types.Context, keeper keeper.Keeper, from string, to []CrossReceiver, symbol string, eventType string, relayerFee int64) {
	if keeper.PbsbServer != nil {
		txHash := ctx.Value(baseapp.TxHashKey)
		if txHashStr, ok := txHash.(string); ok {
			event := CrossTransferEvent{
				TxHash:     txHashStr,
				ChainId:    keeper.DestChainName,
				RelayerFee: relayerFee,
				Type:       eventType,
				From:       from,
				Denom:      symbol,
				To:         to,
			}
			keeper.PbsbServer.Publish(event)
		} else {
			ctx.Logger().With("module", "bridge").Error("failed to get txhash, will not publish cross transfer event ")
		}
	}
}

func publishBindSuccessEvent(ctx types.Context, keeper keeper.Keeper, from string, to []CrossReceiver, symbol string, eventType string, relayerFee int64, contract string, decimals int8) {
	if keeper.PbsbServer != nil {
		txHash := ctx.Value(baseapp.TxHashKey)
		if txHashStr, ok := txHash.(string); ok {
			event := CrossTransferEvent{
				TxHash:     txHashStr,
				ChainId:    keeper.DestChainName,
				RelayerFee: relayerFee,
				Type:       eventType,
				From:       from,
				Denom:      symbol,
				Contract:   contract,
				Decimals:   int(decimals),
				To:         to,
			}
			keeper.PbsbServer.Publish(event)
		} else {
			ctx.Logger().With("module", "bridge").Error("failed to get txhash, will not publish cross transfer event ")
		}
	}
}

type MirrorEvent struct {
	TxHash         string
	ChainId        string
	Type           string
	RelayerFee     int64
	Sender         string
	Contract       string
	BEP20Name      string
	BEP20Symbol    string
	BEP2Symbol     string
	OldTotalSupply int64
	TotalSupply    int64
	Decimals       int
	Fee            int64
}

func (event MirrorEvent) GetTopic() pubsub.Topic {
	return MirrorTopic
}

func publishMirrorEvent(ctx types.Context, keeper keeper.Keeper, pkg *btype.MirrorSynPackage, symbol string, supply int64, fee int64, relayFee int64) {
	if keeper.PbsbServer != nil {
		txHash := ctx.Value(baseapp.TxHashKey)
		if txHashStr, ok := txHash.(string); ok {
			event := MirrorEvent{
				TxHash:      txHashStr,
				ChainId:     keeper.DestChainName,
				Type:        MirrorType,
				RelayerFee:  relayFee,
				Sender:      pkg.MirrorSender.String(),
				Contract:    pkg.ContractAddr.String(),
				BEP20Name:   btype.BytesToSymbol(pkg.BEP20Name),
				BEP20Symbol: btype.BytesToSymbol(pkg.BEP20Symbol),
				BEP2Symbol:  symbol,
				TotalSupply: supply,
				Decimals:    int(pkg.BEP20Decimals),
				Fee:         fee,
			}
			keeper.PbsbServer.Publish(event)
		} else {
			ctx.Logger().With("module", "bridge").Error("failed to get txhash, will not publish mirror event ")
		}
	}
}

func publishMirrorSyncEvent(ctx types.Context, keeper keeper.Keeper, pkg *btype.MirrorSyncSynPackage, symbol string, oldSupply int64, supply int64, fee int64, relayFee int64) {
	if keeper.PbsbServer != nil {
		txHash := ctx.Value(baseapp.TxHashKey)
		if txHashStr, ok := txHash.(string); ok {
			event := MirrorEvent{
				TxHash:         txHashStr,
				ChainId:        keeper.DestChainName,
				Type:           MirrorSyncType,
				RelayerFee:     relayFee,
				Sender:         pkg.SyncSender.String(),
				Contract:       pkg.ContractAddr.String(),
				BEP2Symbol:     symbol,
				OldTotalSupply: oldSupply,
				TotalSupply:    supply,
				Fee:            fee,
			}
			keeper.PbsbServer.Publish(event)
		} else {
			ctx.Logger().With("module", "bridge").Error("failed to get txhash, will not publish mirror sync event ")
		}
	}
}
