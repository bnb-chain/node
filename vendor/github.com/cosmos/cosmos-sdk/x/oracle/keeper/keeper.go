package keeper

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/pubsub"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/oracle/metrics"
	"github.com/cosmos/cosmos-sdk/x/oracle/types"
	pTypes "github.com/cosmos/cosmos-sdk/x/paramHub/types"
	param "github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
)

// Keeper maintains the link to data storage and
// exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc      *codec.Codec
	storeKey sdk.StoreKey

	// The reference to the Paramstore to get and set gov specific params
	paramSpace param.Subspace

	Pool *sdk.Pool

	stakeKeeper types.StakingKeeper
	ScKeeper    sidechain.Keeper
	IbcKeeper   ibc.Keeper
	BkKeeper    bank.Keeper

	Metrics   *metrics.Metrics
	pubServer *pubsub.Server
}

// Parameter store
const (
	DefaultParamSpace = "oracle"
)

func ParamTypeTable() param.TypeTable {
	return param.NewTypeTable().RegisterParamSet(&types.Params{})
}

// NewKeeper creates new instances of the oracle Keeper
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, paramSpace param.Subspace, stakeKeeper types.StakingKeeper,
	scKeeper sidechain.Keeper, ibcKeeper ibc.Keeper, bkKeeper bank.Keeper, pool *sdk.Pool,
) Keeper {
	return Keeper{
		cdc:         cdc,
		storeKey:    storeKey,
		paramSpace:  paramSpace.WithTypeTable(ParamTypeTable()),
		stakeKeeper: stakeKeeper,
		ScKeeper:    scKeeper,
		IbcKeeper:   ibcKeeper,
		BkKeeper:    bkKeeper,
		Metrics:     metrics.NopMetrics(),
		Pool:        pool,
	}
}

func (k Keeper) GetConsensusNeeded(ctx sdk.Context) (consensusNeeded sdk.Dec) {
	k.paramSpace.Get(ctx, types.ParamStoreKeyProphecyParams, &consensusNeeded)
	return
}

func (k *Keeper) EnablePrometheusMetrics() {
	k.Metrics = metrics.PrometheusMetrics()
}

func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

func (k *Keeper) SetPbsbServer(p *pubsub.Server) {
	k.pubServer = p
}

// GetProphecy gets the entire prophecy data struct for a given id
func (k Keeper) GetProphecy(ctx sdk.Context, id string) (types.Prophecy, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(id))
	if bz == nil {
		return types.Prophecy{}, false
	}

	var dbProphecy types.DBProphecy
	k.cdc.MustUnmarshalBinaryBare(bz, &dbProphecy)

	deSerializedProphecy, err := dbProphecy.DeserializeFromDB()
	if err != nil {
		return types.Prophecy{}, false
	}

	return deSerializedProphecy, true
}

// DeleteProphecy delete prophecy for a given id
func (k Keeper) DeleteProphecy(ctx sdk.Context, id string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete([]byte(id))
}

// setProphecy saves a prophecy with an initial claim
func (k Keeper) setProphecy(ctx sdk.Context, prophecy types.Prophecy) {
	store := ctx.KVStore(k.storeKey)
	serializedProphecy, err := prophecy.SerializeForDB()
	if err != nil {
		panic(err)
	}

	store.Set([]byte(prophecy.ID), k.cdc.MustMarshalBinaryBare(serializedProphecy))
}

// ProcessClaim ...
func (k Keeper) ProcessClaim(ctx sdk.Context, claim types.Claim) (types.Prophecy, sdk.Error) {
	activeValidator := k.checkActiveValidator(ctx, claim.ValidatorAddress)
	if !activeValidator {
		return types.Prophecy{}, types.ErrInvalidValidator()
	}

	if claim.ID == "" {
		return types.Prophecy{}, types.ErrInvalidIdentifier()
	}

	if len(claim.Payload) == 0 {
		return types.Prophecy{}, types.ErrInvalidClaim()
	}

	prophecy, found := k.GetProphecy(ctx, claim.ID)
	if !found {
		prophecy = types.NewProphecy(claim.ID)
	}

	switch prophecy.Status.Text {
	case types.PendingStatusText:
		// continue processing
	default:
		return types.Prophecy{}, types.ErrProphecyFinalized()
	}

	prophecy.AddClaim(claim.ValidatorAddress, claim.Payload)
	prophecy = k.processCompletion(ctx, prophecy)

	k.setProphecy(ctx, prophecy)
	return prophecy, nil
}

func (k Keeper) checkActiveValidator(ctx sdk.Context, validatorAddress sdk.ValAddress) bool {
	validator, found := k.stakeKeeper.GetValidator(ctx, validatorAddress)
	if !found {
		return false
	}

	return validator.GetStatus().Equal(sdk.Bonded)
}

// processCompletion looks at a given prophecy
// and assesses whether the claim with the highest power on that prophecy has enough
// power to be considered successful, or alternatively,
// will never be able to become successful due to not enough validation power being
// left to push it over the threshold required for consensus.
func (k Keeper) processCompletion(ctx sdk.Context, prophecy types.Prophecy) types.Prophecy {
	highestClaim, highestClaimPower, totalClaimsPower := prophecy.FindHighestClaim(ctx, k.stakeKeeper)
	totalPower := k.stakeKeeper.GetLastTotalPower(ctx)

	highestConsensusRatio := sdk.NewDec(highestClaimPower).Quo(sdk.NewDec(totalPower))
	remainingPossibleClaimPower := totalPower - totalClaimsPower
	highestPossibleClaimPower := highestClaimPower + remainingPossibleClaimPower

	highestPossibleConsensusRatio := sdk.NewDec(highestPossibleClaimPower).Quo(sdk.NewDec(totalPower))

	consensusNeeded := k.GetConsensusNeeded(ctx)

	if highestConsensusRatio.GTE(consensusNeeded) {
		prophecy.Status.Text = types.SuccessStatusText
		prophecy.Status.FinalClaim = highestClaim
	} else if highestPossibleConsensusRatio.LT(consensusNeeded) {
		prophecy.Status.Text = types.FailedStatusText
	}
	return prophecy
}

func (k *Keeper) SubscribeParamChange(hub pTypes.ParamChangePublisher) {
	hub.SubscribeParamChange(
		func(context sdk.Context, iChange interface{}) {
			switch change := iChange.(type) {
			case *types.Params:
				// do double check
				err := change.UpdateCheck()
				if err != nil {
					context.Logger().Error("skip invalid param change", "err", err, "param", change)
				} else {
					newCtx := context.DepriveSideChainKeyPrefix()
					k.SetParams(newCtx, *change)
					break
				}
			default:
				context.Logger().Debug("skip unknown param change")
			}
		},
		&pTypes.ParamSpaceProto{ParamSpace: k.paramSpace, Proto: func() pTypes.SCParam {
			return new(types.Params)
		}},
		nil,
		nil,
	)
}

func (k *Keeper) PublishCrossAppFailEvent(ctx sdk.Context, from string, relayerFee int64, chainId string) {
	if k.pubServer != nil {
		txHash := ctx.Value(baseapp.TxHashKey)
		if txHashStr, ok := txHash.(string); ok {
			event := types.CrossAppFailEvent{
				TxHash:     txHashStr,
				ChainId:    chainId,
				RelayerFee: relayerFee,
				From:       from,
			}
			k.pubServer.Publish(event)
		} else {
			ctx.Logger().With("module", "oracle").Error("failed to get txhash, will not publish oracle event ")
		}
	}
}
