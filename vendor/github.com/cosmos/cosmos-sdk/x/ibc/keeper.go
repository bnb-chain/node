package ibc

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/cosmos/cosmos-sdk/bsc"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
	param "github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	sTypes "github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

// IBC Keeper
type Keeper struct {
	storeKey  sdk.StoreKey
	codespace sdk.CodespaceType

	paramSpace       param.Subspace
	packageCollector *packageCollector
	sideKeeper       sidechain.Keeper
}

func ParamTypeTable() param.TypeTable {
	return param.NewTypeTable().RegisterParamSet(&Params{})
}

func NewKeeper(storeKey sdk.StoreKey, paramSpace param.Subspace, codespace sdk.CodespaceType, sideKeeper sidechain.Keeper) Keeper {
	return Keeper{
		storeKey:         storeKey,
		codespace:        codespace,
		packageCollector: newPackageCollector(),
		paramSpace:       paramSpace.WithTypeTable(ParamTypeTable()),
		sideKeeper:       sideKeeper,
	}
}

func (k *Keeper) CreateIBCSyncPackage(ctx sdk.Context, destChainName string, channelName string, packageLoad []byte) (uint64, sdk.Error) {
	relayerFee, err := k.GetRelayerFeeParam(ctx, destChainName)
	if err != nil {
		return 0, ErrFeeParamMismatch(DefaultCodespace, fmt.Sprintf("fail to load relayerFee, %v", err))
	}
	return k.CreateRawIBCPackage(ctx, destChainName, channelName, sdk.SynCrossChainPackageType, packageLoad, *relayerFee)
}

func (k *Keeper) CreateRawIBCPackage(ctx sdk.Context, destChainName string, channelName string,
	packageType sdk.CrossChainPackageType, packageLoad []byte, relayerFee big.Int) (uint64, sdk.Error) {

	destChainID, err := k.sideKeeper.GetDestChainID(destChainName)
	if err != nil {
		return 0, sdk.ErrInternal(err.Error())
	}
	channelID, err := k.sideKeeper.GetChannelID(channelName)
	if err != nil {
		return 0, sdk.ErrInternal(err.Error())
	}

	return k.CreateRawIBCPackageByIdWithFee(ctx, destChainID, channelID, packageType, packageLoad, relayerFee)
}

func (k *Keeper) CreateRawIBCPackageById(ctx sdk.Context, destChainID sdk.ChainID, channelID sdk.ChannelID,
	packageType sdk.CrossChainPackageType, packageLoad []byte) (uint64, sdk.Error) {

	destChainName, err := k.sideKeeper.GetDestChainName(destChainID)
	if err != nil {
		return 0, ErrInvalidChainId(DefaultCodespace, "can not find dest chain id")
	}
	relayerFee, err := k.GetRelayerFeeParam(ctx, destChainName)
	if err != nil {
		return 0, ErrFeeParamMismatch(DefaultCodespace, fmt.Sprintf("fail to load relayerFee, %v", err))
	}

	return k.CreateRawIBCPackageByIdWithFee(ctx, destChainID, channelID, packageType, packageLoad, *relayerFee)
}

func (k *Keeper) CreateRawIBCPackageByIdWithFee(ctx sdk.Context, destChainID sdk.ChainID, channelID sdk.ChannelID,
	packageType sdk.CrossChainPackageType, packageLoad []byte, relayerFee big.Int) (uint64, sdk.Error) {

	if packageType == sdk.SynCrossChainPackageType && k.sideKeeper.GetChannelSendPermission(ctx, destChainID, channelID) != sdk.ChannelAllow {
		return 0, ErrWritePackageForbidden(DefaultCodespace, fmt.Sprintf("channel %d is not allowed to write syn package", channelID))
	}

	sequence := k.sideKeeper.GetSendSequence(ctx, destChainID, channelID)
	key := buildIBCPackageKey(k.sideKeeper.GetSrcChainID(), destChainID, channelID, sequence)
	kvStore := ctx.KVStore(k.storeKey)
	if kvStore.Has(key) {
		return 0, ErrDuplicatedSequence(DefaultCodespace, "duplicated sequence")
	}

	// Assemble the package header
	packageHeader := sTypes.EncodePackageHeader(packageType, relayerFee)

	kvStore.Set(key, append(packageHeader, packageLoad...))
	k.sideKeeper.IncrSendSequence(ctx, destChainID, channelID)

	if ctx.IsDeliverTx() {
		k.packageCollector.collectedPackages = append(k.packageCollector.collectedPackages, packageRecord{
			destChainID: destChainID,
			channelID:   channelID,
			sequence:    sequence,
		})
	}

	return sequence, nil
}

func (k *Keeper) GetIBCPackage(ctx sdk.Context, destChainName string, channelName string, sequence uint64) ([]byte, error) {
	destChainID, err := k.sideKeeper.GetDestChainID(destChainName)
	if err != nil {
		return nil, err
	}
	channelID, err := k.sideKeeper.GetChannelID(channelName)
	if err != nil {
		return nil, err
	}
	return k.GetIBCPackageById(ctx, destChainID, channelID, sequence)
}

func (k *Keeper) GetIBCPackageById(ctx sdk.Context, destChainID sdk.ChainID, channelId sdk.ChannelID, sequence uint64) ([]byte, error) {
	kvStore := ctx.KVStore(k.storeKey)
	key := buildIBCPackageKey(k.sideKeeper.GetSrcChainID(), destChainID, channelId, sequence)
	return kvStore.Get(key), nil
}

func (k *Keeper) CleanupIBCPackage(ctx sdk.Context, destChainName string, channelName string, confirmedSequence uint64) {
	destChainID, err := k.sideKeeper.GetDestChainID(destChainName)
	if err != nil {
		return
	}
	channelID, err := k.sideKeeper.GetChannelID(channelName)
	if err != nil {
		return
	}
	prefixKey := buildIBCPackageKeyPrefix(k.sideKeeper.GetSrcChainID(), destChainID, channelID)
	kvStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(kvStore, prefixKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		packageKey := iterator.Key()
		if len(packageKey) != totalPackageKeyLength {
			continue
		}
		sequence := binary.BigEndian.Uint64(packageKey[totalPackageKeyLength-sequenceLength:])
		if sequence > confirmedSequence {
			break
		}
		kvStore.Delete(packageKey)
	}
}

func (k Keeper) GetRelayerFeeParam(ctx sdk.Context, destChainName string) (relaterFee *big.Int, err error) {
	storePrefix := k.sideKeeper.GetSideChainStorePrefix(ctx, destChainName)
	if storePrefix == nil {
		return nil, fmt.Errorf("invalid sideChainId: %s", destChainName)
	}
	sideChainCtx := ctx.WithSideChainKeyPrefix(storePrefix)
	var relayerFeeParam int64
	k.paramSpace.Get(sideChainCtx, ParamRelayerFee, &relayerFeeParam)
	relaterFee = bsc.ConvertBCAmountToBSCAmount(relayerFeeParam)
	return
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

func (k *Keeper) SubscribeParamChange(hub types.ParamChangePublisher) {
	hub.SubscribeParamChange(
		func(context sdk.Context, iChange interface{}) {
			switch change := iChange.(type) {
			case *Params:
				err := change.UpdateCheck()
				if err != nil {
					context.Logger().Error("skip invalid param change", "err", err, "param", change)
				} else {
					k.SetParams(context, *change)
					break
				}
			default:
				context.Logger().Debug("skip unknown param change")
			}
		},
		&types.ParamSpaceProto{ParamSpace: k.paramSpace, Proto: func() types.SCParam {
			return new(Params)
		}},
		nil,
		nil,
	)
}
