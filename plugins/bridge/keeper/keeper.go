package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"

	cmmtypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
	"github.com/binance-chain/node/plugins/tokens/store"
)

// Keeper maintains the link to data storage and
// exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc *codec.Codec // The wire codec for binary encoding/decoding.

	storeKey    sdk.StoreKey
	Pool        *sdk.Pool
	DestChainId string

	OracleKeeper sdk.OracleKeeper
	BankKeeper   bank.Keeper
	TokenMapper  store.Mapper
	IbcKeeper    ibc.Keeper
}

// NewKeeper creates new instances of the bridge Keeper
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, tokenMapper store.Mapper, oracleKeeper sdk.OracleKeeper,
	bankKeeper bank.Keeper, ibcKeeper ibc.Keeper, pool *sdk.Pool, destChainId string) Keeper {
	return Keeper{
		cdc:          cdc,
		storeKey:     storeKey,
		Pool:         pool,
		BankKeeper:   bankKeeper,
		TokenMapper:  tokenMapper,
		IbcKeeper:    ibcKeeper,
		DestChainId:  destChainId,
		OracleKeeper: oracleKeeper,
	}
}

func (k Keeper) RefundTransferIn(ctx sdk.Context, tokenInfo cmmtypes.Token, transferInClaim types.TransferInClaim, refundReason types.RefundReason) (sdk.Tags, sdk.Error) {
	tags := sdk.NewTags(sdk.TagAction, types.ActionTransferInFailed)

	for idx, refundAddr := range transferInClaim.RefundAddresses {
		var calibratedAmount sdk.Int
		if tokenInfo.ContractDecimals >= cmmtypes.TokenDecimals {
			decimals := sdk.NewIntWithDecimal(1, int(tokenInfo.ContractDecimals-cmmtypes.TokenDecimals))
			calibratedAmount = sdk.NewInt(transferInClaim.Amounts[idx]).Mul(decimals)
		} else {
			decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-tokenInfo.ContractDecimals))
			if !sdk.NewInt(transferInClaim.Amounts[idx]).Mod(decimals).IsZero() {
				return nil, types.ErrInvalidAmount("can't calibrate timeout amount")
			}
			calibratedAmount = sdk.NewInt(transferInClaim.Amounts[idx]).Div(decimals)
		}
		transferInFailurePackage, err := types.SerializeTransferInFailurePackage(calibratedAmount,
			transferInClaim.ContractAddress[:], refundAddr[:], refundReason)

		if err != nil {
			return nil, types.ErrSerializePackageFailed(err.Error())
		}

		seq, sdkErr := k.IbcKeeper.CreateIBCPackage(ctx, k.DestChainId, types.RefundChannel, transferInFailurePackage)
		if sdkErr != nil {
			return nil, sdkErr
		}
		tags = tags.AppendTags(sdk.NewTags(
			types.TransferInRefundSequence, []byte(strconv.Itoa(int(seq))),
			types.TransferOutRefundReason, []byte(refundReason.String()),
		))
	}

	return tags, nil
}

func (k Keeper) CreateBindRequest(ctx sdk.Context, req types.BindRequest) sdk.Error {
	key := types.GetBindRequestKey(req.Symbol)

	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get(key)
	if bz != nil {
		return types.ErrBindRequestExists(fmt.Sprintf("bind request of %s already exists", req.Symbol))
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return sdk.ErrInternal(fmt.Sprintf("marshal bind request error, err=%s", err.Error()))
	}

	kvStore.Set(key, reqBytes)
	return nil
}

func (k Keeper) DeleteBindRequest(ctx sdk.Context, symbol string) {
	key := types.GetBindRequestKey(symbol)

	kvStore := ctx.KVStore(k.storeKey)
	kvStore.Delete(key)
}

func (k Keeper) GetBindRequest(ctx sdk.Context, symbol string) (types.BindRequest, sdk.Error) {
	key := types.GetBindRequestKey(symbol)

	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get(key)
	if bz == nil {
		return types.BindRequest{}, types.ErrBindRequestNotExists(fmt.Sprintf("bind request of %s doest not exist", symbol))
	}

	var bindRequest types.BindRequest
	err := json.Unmarshal(bz, &bindRequest)
	if err != nil {
		return types.BindRequest{}, sdk.ErrInternal(fmt.Sprintf("unmarshal bind request error, err=%s", err.Error()))
	}

	return bindRequest, nil
}
