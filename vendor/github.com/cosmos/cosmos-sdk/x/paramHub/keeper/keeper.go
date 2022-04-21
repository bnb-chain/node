package keeper

import (
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	sTypes "github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

var (
	ParamStoreKeyLastFeeChangeProposalID = []byte("lastFeeChangeProposalID")
	ParamStoreKeyFees                    = []byte("fees")

	// for side chain
	ParamStoreKeySCLastParamsChangeProposalID = []byte("SCLastParamsChangeProposalID")
)

const (
	ParamSpace = "paramhub"

	SafeToleratePeriod = 2 * 7 * 24 * 60 * 60 * time.Second // 2 weeks
)

func ParamTypeTable() params.TypeTable {
	return params.NewTypeTable(
		ParamStoreKeyLastFeeChangeProposalID, types.LastProposalID{},
		ParamStoreKeyFees, []types.FeeParam{},
		ParamStoreKeySCLastParamsChangeProposalID, types.LastProposalID{},
	)
}

type Keeper struct {
	params.Keeper
	cdc        *codec.Codec
	paramSpace params.Subspace

	// just for query
	subscriberParamSpace []*types.ParamSpaceProto

	govKeeper *gov.Keeper
	ibcKeeper *ibc.Keeper
	ScKeeper  *sidechain.Keeper

	updateCallbacks  []func(sdk.Context, interface{})
	genesisCallbacks []func(sdk.Context, interface{})
	loadCallBacks    []func(sdk.Context, interface{})
}

func NewKeeper(cdc *codec.Codec, key *sdk.KVStoreKey, tkey *sdk.TransientStoreKey) *Keeper {
	keeper := Keeper{
		Keeper:               params.NewKeeper(cdc, key, tkey),
		cdc:                  cdc,
		updateCallbacks:      make([]func(sdk.Context, interface{}), 0),
		genesisCallbacks:     make([]func(sdk.Context, interface{}), 0),
		subscriberParamSpace: make([]*types.ParamSpaceProto, 0),
	}
	keeper.paramSpace = keeper.Subspace(ParamSpace).WithTypeTable(ParamTypeTable())
	// Add global callback(belongs to no other plugin) here
	keeper.registerFeeParamCallBack()
	keeper.registerCSCParamsCallBack()
	return &keeper
}

func (keeper *Keeper) GetSubscriberParamSpace() []*types.ParamSpaceProto {
	return keeper.subscriberParamSpace
}

func (keeper *Keeper) SetGovKeeper(govKeeper *gov.Keeper) {
	keeper.govKeeper = govKeeper
}

func (keeper *Keeper) SetupForSideChain(scKeeper *sidechain.Keeper, ibcKeeper *ibc.Keeper) {
	keeper.ScKeeper = scKeeper
	keeper.ibcKeeper = ibcKeeper
	keeper.initIbc()
}

func (keeper *Keeper) GetCodeC() *codec.Codec {
	return keeper.cdc
}

func (keeper Keeper) initIbc() {
	if keeper.ibcKeeper == nil {
		return
	}
	err := keeper.ScKeeper.RegisterChannel(ChannelName, ChannelId, &keeper)
	if err != nil {
		panic(fmt.Sprintf("register ibc channel failed, channel=%s, err=%s", ChannelName, err.Error()))
	}
}

func (keeper *Keeper) EndBreatheBlock(ctx sdk.Context) {
	log := keeper.Logger(ctx)
	log.Info("Sync breath block params proposals.")
	feeChange := keeper.getLastFeeChangeParam(ctx)
	if feeChange != nil {
		keeper.notifyOnUpdate(ctx, feeChange)
	}
	if sdk.IsUpgrade(sdk.LaunchBscUpgrade) {
		_, storePrefixes := keeper.ScKeeper.GetAllSideChainPrefixes(ctx)
		for i := range storePrefixes {
			sideChainCtx := ctx.WithSideChainKeyPrefix(storePrefixes[i])
			scParamChanges := keeper.getLastSCParamChanges(sideChainCtx)
			if scParamChanges != nil {
				for _, change := range scParamChanges.SCParams {
					keeper.notifyOnUpdate(sideChainCtx, change)
				}
			}
		}
	}
	return
}

func (keeper *Keeper) EndBlock(ctx sdk.Context) {
	log := keeper.Logger(ctx)
	log.Info("Sync params proposals.")
	if sdk.IsUpgrade(sdk.LaunchBscUpgrade) && keeper.ScKeeper != nil {
		sideChainIds, storePrefixes := keeper.ScKeeper.GetAllSideChainPrefixes(ctx)
		for idx := range storePrefixes {
			sideChainCtx := ctx.WithSideChainKeyPrefix(storePrefixes[idx])
			cscChanges := keeper.getLastCSCParamChanges(sideChainCtx)
			if len(cscChanges) > 0 {
				keeper.notifyOnUpdate(sideChainCtx, types.CSCParamChanges{Changes: cscChanges, ChainID: sideChainIds[idx]})
			}
		}
	}
	return
}

func (keeper *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "paramHub")
}

func (keeper *Keeper) notifyOnUpdate(context sdk.Context, change interface{}) {
	for _, c := range keeper.updateCallbacks {
		c(context, change)
	}
}

func (keeper *Keeper) notifyOnLoad(ctx sdk.Context, load interface{}) {
	for _, c := range keeper.loadCallBacks {
		c(ctx, load)
	}
}

func (keeper *Keeper) notifyOnGenesis(ctx sdk.Context, state interface{}) {
	for _, c := range keeper.genesisCallbacks {
		c(ctx, state)
	}
}

func (keeper *Keeper) InitGenesis(ctx sdk.Context, params types.GenesisState) {
	keeper.initFeeGenesis(ctx, params)
	keeper.notifyOnGenesis(ctx, params)
	keeper.setLastFeeChangeProposalId(ctx, types.LastProposalID{0})
}

func (keeper *Keeper) Load(ctx sdk.Context) {
	keeper.loadFeeParam(ctx)
}

func (keeper *Keeper) SubscribeParamChange(updateCb func(sdk.Context, interface{}), spaceProto *types.ParamSpaceProto, genesisCb func(sdk.Context, interface{}), loadCb func(sdk.Context, interface{})) {
	if updateCb != nil {
		keeper.SubscribeUpdateEvent(updateCb)
	}
	if genesisCb != nil {
		keeper.SubscribeGenesisEvent(genesisCb)
	}
	if loadCb != nil {
		keeper.SubscribeLoadEvent(loadCb)
	}
	if spaceProto != nil {
		keeper.subscriberParamSpace = append(keeper.subscriberParamSpace, spaceProto)
	}
}

func (keeper *Keeper) SubscribeUpdateEvent(c func(sdk.Context, interface{})) {
	keeper.updateCallbacks = append(keeper.updateCallbacks, c)
}

func (keeper *Keeper) SubscribeGenesisEvent(c func(sdk.Context, interface{})) {
	keeper.genesisCallbacks = append(keeper.genesisCallbacks, c)
}

func (keeper *Keeper) SubscribeLoadEvent(c func(sdk.Context, interface{})) {
	keeper.loadCallBacks = append(keeper.loadCallBacks, c)
}

// implement cross chain app
func (keeper *Keeper) ExecuteSynPackage(ctx sdk.Context, payload []byte, _ int64) sdk.ExecuteResult {
	panic("receive unexpected package")
}

func (keeper *Keeper) ExecuteAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	var ackPackage sTypes.CommonAckPackage
	err := rlp.DecodeBytes(payload, &ackPackage)
	if err != nil {
		keeper.Logger(ctx).Error("fail to decode ack package", "payload", payload)
		return sdk.ExecuteResult{Err: types.ErrInvalidCrossChainPackage(types.DefaultCodespace)}
	}
	if !ackPackage.IsOk() {
		keeper.Logger(ctx).Error("side chain failed to process param package", "code", ackPackage.Code)
	}
	return sdk.ExecuteResult{}
}

// When the ack application crash, payload is the payload of the origin package.
func (keeper *Keeper) ExecuteFailAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	//do no thing
	keeper.Logger(ctx).Error("side chain process params package crashed", "payload", payload)
	return sdk.ExecuteResult{}
}
