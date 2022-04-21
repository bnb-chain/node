package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/pubsub"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	pTypes "github.com/cosmos/cosmos-sdk/x/paramHub/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	sTypes "github.com/cosmos/cosmos-sdk/x/sidechain/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	"github.com/tendermint/tendermint/libs/log"
)

// keeper of the stake store
type Keeper struct {
	storeKey       sdk.StoreKey
	rewardStoreKey sdk.StoreKey
	storeTKey      sdk.StoreKey
	cdc            *codec.Codec
	bankKeeper     bank.Keeper
	addrPool       *sdk.Pool
	hooks          sdk.StakingHooks
	paramstore     params.Subspace

	// codespace
	codespace sdk.CodespaceType

	// the two keepers are optional,
	// if you want to enable side chains, you need call `SetupForSideChain`
	ibcKeeper *ibc.Keeper
	ScKeeper  *sidechain.Keeper

	PbsbServer *pubsub.Server
}

func NewKeeper(cdc *codec.Codec, key, rewardKey, tkey sdk.StoreKey, ck bank.Keeper, addrPool *sdk.Pool,
	paramstore params.Subspace, codespace sdk.CodespaceType) Keeper {
	keeper := Keeper{
		storeKey:       key,
		rewardStoreKey: rewardKey,
		storeTKey:      tkey,
		cdc:            cdc,
		bankKeeper:     ck,
		addrPool:       addrPool,
		paramstore:     paramstore.WithTypeTable(ParamTypeTable()),
		hooks:          nil,
		codespace:      codespace,
	}

	return keeper
}

func (k Keeper) initIbc() {
	if k.ibcKeeper == nil {
		return
	}
	err := k.ScKeeper.RegisterChannel(ChannelName, ChannelId, &k)
	if err != nil {
		panic(fmt.Sprintf("register ibc channel failed, channel=%s, err=%s", ChannelName, err.Error()))
	}
}

func (k *Keeper) SetupForSideChain(scKeeper *sidechain.Keeper, ibcKeeper *ibc.Keeper) {
	k.ScKeeper = scKeeper
	k.ibcKeeper = ibcKeeper
	k.initIbc()
}

func (k *Keeper) SetPbsbServer(server *pubsub.Server) {
	k.PbsbServer = server
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/stake")
}

// Set the validator hooks
func (k Keeper) WithHooks(sh sdk.StakingHooks) Keeper {
	if k.hooks != nil {
		panic("cannot set validator hooks twice")
	}
	k.hooks = sh
	return k
}

//_________________________________________________________________________

// return the codespace
func (k Keeper) Codespace() sdk.CodespaceType {
	return k.codespace
}

//_______________________________________________________________________

// load the pool
func (k Keeper) GetPool(ctx sdk.Context) (pool types.Pool) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(PoolKey)
	if b == nil {
		panic("stored pool should not have been nil")
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &pool)
	return
}

// set the pool
func (k Keeper) SetPool(ctx sdk.Context, pool types.Pool) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshalBinaryLengthPrefixed(pool)
	store.Set(PoolKey, b)
}

//_______________________________________________________________________

// Load the last total validator power.
func (k Keeper) GetLastTotalPower(ctx sdk.Context) (power int64) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(LastTotalPowerKey)
	if b == nil {
		return 0
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &power)
	return
}

// Set the last total validator power.
func (k Keeper) SetLastTotalPower(ctx sdk.Context, power int64) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshalBinaryLengthPrefixed(power)
	store.Set(LastTotalPowerKey, b)
}

//_______________________________________________________________________

// Load the last validator power.
// Returns zero if the operator was not a validator last block.
func (k Keeper) GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) (power int64) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(GetLastValidatorPowerKey(operator))
	if bz == nil {
		return 0
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &power)
	return
}

// Set the last validator power.
func (k Keeper) SetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress, power int64) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(power)
	store.Set(GetLastValidatorPowerKey(operator), bz)
}

// Delete the last validator power.
func (k Keeper) DeleteLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(GetLastValidatorPowerKey(operator))
}

//__________________________________________________________________________

// get the current in-block validator operation counter
func (k Keeper) GetIntraTxCounter(ctx sdk.Context) int16 {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(IntraTxCounterKey)
	if b == nil {
		return 0
	}
	var counter int16
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &counter)
	return counter
}

// set the current in-block validator operation counter
func (k Keeper) SetIntraTxCounter(ctx sdk.Context, counter int16) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(counter)
	store.Set(IntraTxCounterKey, bz)
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
					res := k.GetParams(context)
					// ignore BondDenom update if have.
					change.BondDenom = res.BondDenom
					k.SetParams(context, *change)
					break
				}

			default:
				context.Logger().Debug("skip unknown param change")
			}
		},
		&pTypes.ParamSpaceProto{ParamSpace: k.paramstore, Proto: func() pTypes.SCParam {
			return new(types.Params)
		}},
		nil,
		nil,
	)
}

// cross chain app implement
func (k *Keeper) ExecuteSynPackage(ctx sdk.Context, payload []byte, _ int64) sdk.ExecuteResult {
	panic("receive unexpected syn package")
}

func (k *Keeper) ExecuteAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	logger := ctx.Logger().With("module", "stake")
	var ackPackage sTypes.CommonAckPackage
	err := rlp.DecodeBytes(payload, &ackPackage)
	if err != nil {
		logger.Error("fail to decode ack package", "payload", payload)
		return sdk.ExecuteResult{Err: types.ErrInvalidCrosschainPackage(k.codespace)}
	}
	if !ackPackage.IsOk() {
		logger.Error("side chain failed to process staking package", "code", ackPackage.Code)
	}
	return sdk.ExecuteResult{}
}

func (k *Keeper) ExecuteFailAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	//do no thing
	ctx.Logger().Error("side chain process staking package crashed", "payload", payload)
	return sdk.ExecuteResult{}
}
