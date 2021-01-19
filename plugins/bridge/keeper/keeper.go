package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/pubsub"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/sidechain"

	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/bridge/types"
	"github.com/binance-chain/node/plugins/tokens/store"
)

// Keeper maintains the link to data storage and
// exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc *codec.Codec // The wire codec for binary encoding/decoding.

	storeKey      sdk.StoreKey
	Pool          *sdk.Pool
	DestChainId   sdk.ChainID
	DestChainName string

	ScKeeper      sidechain.Keeper
	BankKeeper    bank.Keeper
	TokenMapper   store.Mapper
	AccountKeeper auth.AccountKeeper
	IbcKeeper     ibc.Keeper

	PbsbServer *pubsub.Server
}

// NewKeeper creates new instances of the bridge Keeper
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, accountKeeper auth.AccountKeeper, tokenMapper store.Mapper, scKeeper sidechain.Keeper,
	bankKeeper bank.Keeper, ibcKeeper ibc.Keeper, pool *sdk.Pool, destChainId sdk.ChainID, destChainName string) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		Pool:          pool,
		BankKeeper:    bankKeeper,
		TokenMapper:   tokenMapper,
		AccountKeeper: accountKeeper,
		IbcKeeper:     ibcKeeper,
		DestChainId:   destChainId,
		DestChainName: destChainName,
		ScKeeper:      scKeeper,
	}
}

func (k Keeper) RefundTransferIn(decimals int8, transferInClaim *types.TransferInSynPackage, refundReason types.RefundReason) ([]byte, sdk.Error) {
	refundBscAmounts := make([]*big.Int, 0, len(transferInClaim.RefundAddresses))
	for idx := range transferInClaim.RefundAddresses {
		bscAmount, sdkErr := types.ConvertBCAmountToBSCAmount(decimals, transferInClaim.Amounts[idx].Int64())
		if sdkErr != nil {
			return nil, sdkErr
		}

		refundBscAmounts = append(refundBscAmounts, bscAmount.BigInt())
	}

	refundPackage := &types.TransferInRefundPackage{
		ContractAddr:    transferInClaim.ContractAddress,
		RefundAddresses: transferInClaim.RefundAddresses,
		RefundAmounts:   refundBscAmounts,
		RefundReason:    refundReason,
	}

	encodedBytes, err := rlp.EncodeToBytes(refundPackage)
	if err != nil {
		return nil, sdk.ErrInternal("encode refund package error")
	}
	return encodedBytes, nil
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

func (k Keeper) SetContractDecimals(ctx sdk.Context, contractAddr types.SmartChainAddress, decimals int8) {
	key := types.GetContractDecimalsKey(contractAddr[:])

	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get(key)
	if bz != nil {
		return
	}

	kvStore.Set(key, []byte{byte(decimals)})
}

func (k Keeper) GetContractDecimals(ctx sdk.Context, contractAddr types.SmartChainAddress) int8 {
	if sdk.IsUpgrade(upgrade.FixFailAckPackage) {
		if strings.ToLower(contractAddr.String()) == types.BNBContractAddr {
			return types.BNBContractDecimals
		}
	}

	key := types.GetContractDecimalsKey(contractAddr[:])

	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get(key)
	if bz == nil {
		return -1
	}

	return int8(bz[0])
}

func (k *Keeper) SetPbsbServer(server *pubsub.Server) {
	k.PbsbServer = server
}
