package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/x/ibc"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	cmmtypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
	"github.com/binance-chain/node/plugins/oracle"
	"github.com/binance-chain/node/plugins/tokens/store"
)

// Keeper maintains the link to data storage and
// exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc *codec.Codec // The wire codec for binary encoding/decoding.

	oracleKeeper oracle.Keeper

	storeKey sdk.StoreKey // The key used to access the store from the Context.

	SourceChainId uint16
	DestChainId   uint16

	// The reference to the CoinKeeper to modify balances
	BankKeeper bank.Keeper

	TokenMapper store.Mapper

	IbcKeeper ibc.Keeper
}

// NewKeeper creates new instances of the bridge Keeper
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, tokenMapper store.Mapper, oracleKeeper oracle.Keeper,
	bankKeeper bank.Keeper, ibcKeeper ibc.Keeper, sourceChainId, destChainId uint16) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		BankKeeper:    bankKeeper,
		TokenMapper:   tokenMapper,
		IbcKeeper:     ibcKeeper,
		SourceChainId: sourceChainId,
		DestChainId:   destChainId,
		oracleKeeper:  oracleKeeper,
	}
}

func (k Keeper) IncreaseSequence(ctx sdk.Context, key string) {
	currentSequence := k.GetCurrentSequence(ctx, key)

	kvStore := ctx.KVStore(k.storeKey)
	kvStore.Set([]byte(key), []byte(strconv.FormatInt(currentSequence+1, 10)))
}

func (k Keeper) GetCurrentSequence(ctx sdk.Context, key string) int64 {
	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get([]byte(key))
	if bz == nil {
		return types.StartSequence
	}

	sequence, err := strconv.ParseInt(string(bz), 10, 64)
	if err != nil {
		panic(fmt.Errorf("wrong sequence, key=%s, sequence=%s", key, string(bz)))
	}
	return sequence
}

func (k Keeper) ProcessTransferInClaim(ctx sdk.Context, claim oracle.Claim) (oracle.Prophecy, sdk.Tags, sdk.Error) {
	prophecy, err := k.oracleKeeper.ProcessClaim(ctx, claim)
	if err != nil {
		return oracle.Prophecy{}, nil, err
	}

	if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
		return prophecy, nil, nil
	}

	if prophecy.Status.Text != oracle.SuccessStatusText {
		return prophecy, nil, nil
	}

	transferInClaim, err := types.GetTransferInClaimFromOracleClaim(prophecy.Status.FinalClaim)
	if err != nil {
		return oracle.Prophecy{}, nil, err
	}

	tokenInfo, errMsg := k.TokenMapper.GetToken(ctx, transferInClaim.Symbol)
	if errMsg != nil {
		return oracle.Prophecy{}, nil, sdk.ErrInternal(errMsg.Error())
	}

	if transferInClaim.ExpireTime < ctx.BlockHeader().Time.Unix() {
		//timeOutPackage, err := types.SerializeTimeoutPackage(calibratedAmount,
		//	transferInClaim.ContractAddress[:], transferInClaim.RefundAddresses[:])
		//
		//if err != nil {
		//	return oracle.Prophecy{}, nil, types.ErrSerializePackageFailed(err.Error())
		//}
		//
		//timeoutChannelId, err := sdk.GetChannelID(types.TimeoutChannelName)
		//if err != nil {
		//	return oracle.Prophecy{}, nil, types.ErrGetChannelIdFailed(err.Error())
		//}
		//
		//transferInTimeoutSequence := k.IbcKeeper.GetNextSequence(ctx, sdk.CrossChainID(k.DestChainId), timeoutChannelId)
		//sdkErr := k.IbcKeeper.CreateIBCPackage(ctx, sdk.CrossChainID(k.DestChainId), timeoutChannelId, timeOutPackage)
		//if sdkErr != nil {
		//	return oracle.Prophecy{}, nil, sdkErr
		//}
		//tags := sdk.NewTags(
		//	sdk.TagAction, types.ActionTransferInTimeOut,
		//	types.TransferInTimeoutSequence, []byte(strconv.Itoa(int(transferInTimeoutSequence))),
		//)
		//
		//return prophecy, tags, nil
	}

	for idx, receiverAddr := range transferInClaim.ReceiverAddresses {
		var calibratedAmount int64
		if cmmtypes.TokenDecimals > tokenInfo.ContractDecimals {
			decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-tokenInfo.ContractDecimals))
			calibratedAmount = transferInClaim.Amounts[idx].Mul(decimals).Int64()
		} else {
			decimals := sdk.NewIntWithDecimal(1, int(tokenInfo.ContractDecimals-cmmtypes.TokenDecimals))
			calibratedAmount = transferInClaim.Amounts[idx].Div(decimals).Int64()
		}

		_, err = k.BankKeeper.SendCoins(ctx, types.PegAccount, receiverAddr,
			sdk.Coins{sdk.Coin{Denom: transferInClaim.Symbol, Amount: calibratedAmount}})
		if err != nil {
			return oracle.Prophecy{}, nil, err
		}
	}

	// TODO distribute delay fee

	// TODO should we delete prophecy when prophecy succeeds

	// increase sequence
	k.IncreaseSequence(ctx, types.KeyCurrentTransferInSequence)
	return prophecy, nil, nil
}

func (k Keeper) ProcessUpdateTransferOutClaim(ctx sdk.Context, claim oracle.Claim) (oracle.Prophecy, sdk.Error) {
	prophecy, err := k.oracleKeeper.ProcessClaim(ctx, claim)
	if err != nil {
		return oracle.Prophecy{}, err
	}

	if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
		return prophecy, nil
	}
	if prophecy.Status.Text != oracle.SuccessStatusText {
		return prophecy, nil
	}

	updateTransferOutClaim, err := types.GetUpdateTransferOutClaimFromOracleClaim(prophecy.Status.FinalClaim)
	if err != nil {
		return oracle.Prophecy{}, err
	}

	_, err = k.BankKeeper.SendCoins(ctx, types.PegAccount, updateTransferOutClaim.SenderAddress, sdk.Coins{updateTransferOutClaim.Amount})
	if err != nil {
		return oracle.Prophecy{}, err
	}

	k.IncreaseSequence(ctx, types.KeyUpdateTransferOutSequence)

	return prophecy, nil
}

func (k Keeper) ProcessUpdateBindClaim(ctx sdk.Context, claim oracle.Claim) (oracle.Prophecy, sdk.Error) {
	prophecy, err := k.oracleKeeper.ProcessClaim(ctx, claim)
	if err != nil {
		return oracle.Prophecy{}, err
	}

	if prophecy.Status.Text == oracle.SuccessStatusText {
		updateBindClaim, err := types.GetUpdateBindClaimFromOracleClaim(prophecy.Status.FinalClaim)
		if err != nil {
			return oracle.Prophecy{}, err
		}

		bindRequest, err := k.GetBindRequest(ctx, updateBindClaim.Symbol)
		if err != nil {
			return oracle.Prophecy{}, err
		}

		if bindRequest.Symbol != updateBindClaim.Symbol ||
			!bindRequest.Amount.Equal(updateBindClaim.Amount) ||
			bindRequest.ContractAddress.String() != updateBindClaim.ContractAddress.String() ||
			bindRequest.ContractDecimals != updateBindClaim.ContractDecimals {

			return oracle.Prophecy{}, types.ErrBindRequestNotIdentical("update bind claim is not identical to bind request")
		}

		if updateBindClaim.Status == types.BindStatusSuccess {
			stdError := k.TokenMapper.UpdateBind(ctx, updateBindClaim.Symbol,
				updateBindClaim.ContractAddress.String(), updateBindClaim.ContractDecimals)

			if stdError != nil {
				return oracle.Prophecy{}, sdk.ErrInternal(fmt.Sprintf("update token bind info error"))
			}
		} else {
			var calibratedAmount int64
			if cmmtypes.TokenDecimals > bindRequest.ContractDecimals {
				decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-bindRequest.ContractDecimals))
				calibratedAmount = bindRequest.Amount.Mul(decimals).Int64()
			} else {
				decimals := sdk.NewIntWithDecimal(1, int(bindRequest.ContractDecimals-cmmtypes.TokenDecimals))
				calibratedAmount = bindRequest.Amount.Div(decimals).Int64()
			}
			_, err = k.BankKeeper.SendCoins(ctx, types.PegAccount, bindRequest.From,
				sdk.Coins{sdk.Coin{Denom: bindRequest.Symbol, Amount: calibratedAmount}})
			if err != nil {
				return oracle.Prophecy{}, err
			}
		}

		k.DeleteBindRequest(ctx, updateBindClaim.Symbol)

		// TODO Distribute fee
		k.IncreaseSequence(ctx, types.KeyUpdateBindSequence)
	} else if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	}
	return prophecy, nil
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
