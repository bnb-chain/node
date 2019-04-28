package paramhub

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/params"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/plugins/param/types"
)

var (
	ParamStoreKeyLastFeeChangeProposalID = []byte("lastFeeChangeProposalID")
	ParamStoreKeyFees                    = []byte("fees")
	//Add other parameter store key here
)

const (
	DefaultParamSpace = "paramhub"
)

func ParamTypeTable() params.TypeTable {
	return params.NewTypeTable(
		ParamStoreKeyLastFeeChangeProposalID, types.LastProposalID{},
		ParamStoreKeyFees, []types.FeeParam{},
	)
}

type Keeper struct {
	params.Keeper
	cdc        *codec.Codec
	paramSpace params.Subspace
	codespace  sdk.CodespaceType

	govKeeper gov.Keeper

	updateCallbacks  []func(sdk.Context, []interface{})
	genesisCallbacks []func(sdk.Context, types.GenesisState)
	loadCallBacks    []func(sdk.Context, interface{})

	logger tmlog.Logger
}

func NewKeeper(cdc *codec.Codec, key *sdk.KVStoreKey, tkey *sdk.TransientStoreKey) *Keeper {
	logger := bnclog.With("module", "paramHub")
	keeper := Keeper{
		Keeper:           params.NewKeeper(cdc, key, tkey),
		cdc:              cdc,
		updateCallbacks:  make([]func(sdk.Context, []interface{}), 0),
		genesisCallbacks: make([]func(sdk.Context, types.GenesisState), 0),
		logger:           logger,
	}
	keeper.paramSpace = keeper.Subspace(DefaultParamSpace).WithTypeTable(ParamTypeTable())
	// Add global callback(belongs to no other plugin) here
	keeper.registerFeeParamCallBack()
	return &keeper
}

func (keeper *Keeper) SetGovKeeper(govKeeper gov.Keeper) {
	keeper.govKeeper = govKeeper
}

func (keeper *Keeper) EndBreatheBlock(ctx sdk.Context) {
	keeper.logger.Info("Sync params proposals.")
	changes := make([]interface{}, 0)
	feeChange := keeper.getLastFeeChangeParam(ctx)
	if feeChange != nil {
		changes = append(changes, feeChange)
	}
	// Add other param change here
	if len(changes) != 0 {
		keeper.notifyOnUpdate(ctx, changes)
	}
	return
}

func (keeper *Keeper) notifyOnUpdate(ctx sdk.Context, changes []interface{}) {
	for _, c := range keeper.updateCallbacks {
		c(ctx, changes)
	}
}

func (keeper *Keeper) notifyOnLoad(ctx sdk.Context, load interface{}) {
	for _, c := range keeper.loadCallBacks {
		c(ctx, load)
	}
}

func (keeper *Keeper) notifyOnGenesis(ctx sdk.Context, state types.GenesisState) {
	for _, c := range keeper.genesisCallbacks {
		c(ctx, state)
	}
}

func (keeper *Keeper) InitGenesis(ctx sdk.Context, params types.GenesisState) {
	keeper.initFeeGenesis(ctx, params)
	keeper.notifyOnGenesis(ctx, params)
	keeper.setLastFeeChangeProposalId(ctx, types.LastProposalID{0})
}

// For fee parameters
func (keeper *Keeper) getLastFeeChangeProposalId(ctx sdk.Context) types.LastProposalID {
	var id types.LastProposalID
	keeper.paramSpace.Get(ctx, ParamStoreKeyLastFeeChangeProposalID, &id)
	return id
}

func (keeper *Keeper) setLastFeeChangeProposalId(ctx sdk.Context, id types.LastProposalID) {
	keeper.paramSpace.Set(ctx, ParamStoreKeyLastFeeChangeProposalID, &id)
	return
}

func (keeper *Keeper) getLastFeeChangeParam(ctx sdk.Context) []types.FeeParam {
	var latestProposal *gov.Proposal
	lastProposalId := keeper.getLastFeeChangeProposalId(ctx)
	keeper.govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, lastProposalId.ProposalID, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeFeeChange {
			latestProposal = &proposal
			return true
		}
		return false
	})
	if latestProposal != nil {
		var changeParam types.FeeChangeParams
		strProposal := (*latestProposal).GetDescription()
		err := keeper.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
		if err != nil {
			panic(fmt.Sprintf("Get broken data when unmarshal FeeChangeParams msg. %v", err))
		}
		// setLastFeeProposal first. If invalid, the proposal before it will not been processed too.
		keeper.setLastFeeChangeProposalId(ctx, types.LastProposalID{(*latestProposal).GetProposalID()})
		if err := changeParam.Check(); err != nil {
			keeper.logger.Error("The latest fee param change proposal is invalid.", "proposalId", (*latestProposal).GetProposalID(), "param", changeParam, "err", err)
			return nil
		}
		return changeParam.FeeParams
	}
	return nil
}

func (keeper *Keeper) Load(ctx sdk.Context) {
	keeper.loadFeeParam(ctx)
	// Add other param load here
}

func (keeper *Keeper) SubscribeParamChange(u func(sdk.Context, []interface{}), g func(sdk.Context, types.GenesisState), l func(sdk.Context, interface{})) {
	keeper.SubscribeUpdateEvent(u)
	keeper.SubscribeGenesisEvent(g)
	keeper.SubscribeLoadEvent(l)
}

func (keeper *Keeper) SubscribeUpdateEvent(c func(sdk.Context, []interface{})) {
	keeper.updateCallbacks = append(keeper.updateCallbacks, c)
}

func (keeper *Keeper) SubscribeGenesisEvent(c func(sdk.Context, types.GenesisState)) {
	keeper.genesisCallbacks = append(keeper.genesisCallbacks, c)
}

func (keeper *Keeper) SubscribeLoadEvent(c func(sdk.Context, interface{})) {
	keeper.loadCallBacks = append(keeper.loadCallBacks, c)
}
