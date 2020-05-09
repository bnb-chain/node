package paramhub

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/x/params/subspace"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/plugins/param/types"
)

var (
	ParamStoreKeyLastFeeChangeProposalID = []byte("lastFeeChangeProposalID")
	ParamStoreKeyFees                    = []byte("fees")

	// for side chain
	ParamStoreKeyCSCLastParamsChangeProposalID = []byte("CSCLastParamsChangeProposalID")
	ParamStoreKeySCLastParamsChangeProposalID  = []byte("SCLastParamsChangeProposalID")
)

const (
	NativeParamSpace = "paramhub"
	SideParamSpace   = "sideParamHub"
)

func NativeParamTypeTable() params.TypeTable {
	return params.NewTypeTable(
		ParamStoreKeyLastFeeChangeProposalID, types.LastProposalID{},
		ParamStoreKeyFees, []types.FeeParam{},
	)
}

func SideParamTypeTable() params.TypeTable {
	return params.NewTypeTable(
		ParamStoreKeyCSCLastParamsChangeProposalID, types.LastProposalID{},
		ParamStoreKeySCLastParamsChangeProposalID, types.LastProposalID{},
	)
}

type Keeper struct {
	params.Keeper
	cdc              *codec.Codec
	nativeParamSpace params.Subspace
	sideParamSpace   params.Subspace
	codespace        sdk.CodespaceType

	// just for query
	subscriberParamSpace []subspace.SubParamSpaceKey

	govKeeper *gov.Keeper
	ibcKeeper *ibc.Keeper
	scKeeper  *sidechain.Keeper

	updateCallbacks  []func([]sdk.Context, []interface{})
	genesisCallbacks []func(sdk.Context, interface{})
	loadCallBacks    []func(sdk.Context, interface{})

	logger tmlog.Logger
}

func NewKeeper(cdc *codec.Codec, key *sdk.KVStoreKey, tkey *sdk.TransientStoreKey) *Keeper {
	logger := bnclog.With("module", "paramHub")
	keeper := Keeper{
		Keeper:           params.NewKeeper(cdc, key, tkey),
		cdc:              cdc,
		updateCallbacks:  make([]func([]sdk.Context, []interface{}), 0),
		genesisCallbacks: make([]func(sdk.Context, interface{}), 0),
		logger:           logger,
	}
	keeper.nativeParamSpace = keeper.Subspace(NativeParamSpace).WithTypeTable(NativeParamTypeTable())
	keeper.sideParamSpace = keeper.Subspace(SideParamSpace).WithTypeTable(SideParamTypeTable())
	// Add global callback(belongs to no other plugin) here
	keeper.registerFeeParamCallBack()
	keeper.registerCSCParamsCallBack()
	return &keeper
}

func (keeper *Keeper) SetGovKeeper(govKeeper *gov.Keeper) {
	keeper.govKeeper = govKeeper
}

func (keeper *Keeper) SetupForSideChain(scKeeper *sidechain.Keeper, ibcKeeper *ibc.Keeper) {
	keeper.scKeeper = scKeeper
	keeper.ibcKeeper = ibcKeeper
	keeper.initIbc()
}

func (keeper Keeper) initIbc() {
	if keeper.ibcKeeper == nil {
		return
	}
	err := keeper.ibcKeeper.RegisterChannel(IbcChannelName, IbcChannelId)
	if err != nil {
		panic(fmt.Sprintf("register ibc channel failed, channel=%s, err=%s", IbcChannelName, err.Error()))
	}
}

func (keeper *Keeper) EndBreatheBlock(ctx sdk.Context) {
	keeper.logger.Info("Sync breath block params proposals.")
	changes := make([]interface{}, 0)
	contexts := make([]sdk.Context, 0)
	feeChange := keeper.getLastFeeChangeParam(ctx)
	if feeChange != nil {
		changes = append(changes, feeChange)
		contexts = append(contexts, ctx)
	}
	if sdk.IsUpgrade(sdk.LaunchBscUpgrade) {
		_, storePrefixes := keeper.scKeeper.GetAllSideChainPrefixes(ctx)
		scChangeItems := make([]types.SCParam, 0)
		scChangeContexts := make([]sdk.Context, 0)
		for i := range storePrefixes {
			sideChainCtx := ctx.WithSideChainKeyPrefix(storePrefixes[i])
			scParamChanges := keeper.getLastSCParamChanges(sideChainCtx)
			for _, change := range scParamChanges {
				if err := change.Check(); err != nil {
					keeper.logger.Error("scParamChanges check failed will,skip", "scParamChanges", change, "err", err)
					continue
				}
				for _, c := range change.SCParams {
					scChangeItems = append(scChangeItems, c)
					if _, native, _ := c.GetParamAttribute(); native {
						scChangeContexts = append(scChangeContexts, ctx)
					} else {
						scChangeContexts = append(scChangeContexts, sideChainCtx)
					}
				}
			}
		}
		// reverse
		for j := len(scChangeItems) - 1; j >= 0; j-- {
			changes = append(changes, scChangeItems[j].Value())
			contexts = append(contexts, scChangeContexts[j])
		}
	}
	if len(changes) != 0 {
		keeper.notifyOnUpdate(contexts, changes)
	}
	return
}

func (keeper *Keeper) EndBlock(ctx sdk.Context) {
	keeper.logger.Info("Sync params proposals.")
	changes := make([]interface{}, 0)
	contexts := make([]sdk.Context, 0)
	if sdk.IsUpgrade(sdk.LaunchBscUpgrade) && keeper.scKeeper != nil {
		sideChainIds, storePrefixes := keeper.scKeeper.GetAllSideChainPrefixes(ctx)
		for i := range storePrefixes {
			sideChainCtx := ctx.WithSideChainKeyPrefix(storePrefixes[i])
			cscChanges := keeper.getLastCSCParamChanges(sideChainCtx)
			if len(cscChanges) > 0 {
				changes = append(changes, types.CSCParamChanges{Changes: cscChanges, ChainID: sideChainIds[i]})
				contexts = append(contexts, ctx)
			}
		}
	}
	if len(changes) != 0 {
		keeper.notifyOnUpdate(contexts, changes)
	}
	return
}

func (keeper *Keeper) notifyOnUpdate(contexts []sdk.Context, changes []interface{}) {
	for _, c := range keeper.updateCallbacks {
		c(contexts, changes)
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

func (keeper *Keeper) SubscribeParamChange(u func([]sdk.Context, []interface{}), s *subspace.SubParamSpaceKey, g func(sdk.Context, interface{}), l func(sdk.Context, interface{})) {
	if u != nil {
		keeper.SubscribeUpdateEvent(u)
	}
	if g != nil {
		keeper.SubscribeGenesisEvent(g)
	}
	if l != nil {
		keeper.SubscribeLoadEvent(l)
	}
}

func (keeper *Keeper) SubscribeUpdateEvent(c func([]sdk.Context, []interface{})) {
	keeper.updateCallbacks = append(keeper.updateCallbacks, c)
}

func (keeper *Keeper) SubscribeGenesisEvent(c func(sdk.Context, interface{})) {
	keeper.genesisCallbacks = append(keeper.genesisCallbacks, c)
}

func (keeper *Keeper) SubscribeLoadEvent(c func(sdk.Context, interface{})) {
	keeper.loadCallBacks = append(keeper.loadCallBacks, c)
}