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

	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(key), []byte(strconv.FormatInt(currentSequence+1, 10)))
}

func (k Keeper) GetCurrentSequence(ctx sdk.Context, key string) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(key))
	if bz == nil {
		return types.StartSequence
	}

	sequence, err := strconv.ParseInt(string(bz), 10, 64)
	if err != nil {
		panic(fmt.Errorf("wrong sequence, key=%s, sequence=%s", key, string(bz)))
	}
	return sequence
}

func (k Keeper) ProcessTransferClaim(ctx sdk.Context, claim oracle.Claim) (oracle.Prophecy, sdk.Error) {
	prophecy, err := k.oracleKeeper.ProcessClaim(ctx, claim)
	if err != nil {
		return oracle.Prophecy{}, err
	}

	if prophecy.Status.Text == oracle.SuccessStatusText {
		transferClaim, err := types.GetTransferClaimFromOracleClaim(prophecy.Status.FinalClaim)
		if err != nil {
			return oracle.Prophecy{}, err
		}

		tokenInfo, errMsg := k.TokenMapper.GetToken(ctx, transferClaim.Amount.Denom)
		if errMsg != nil {
			return oracle.Prophecy{}, sdk.ErrInternal(errMsg.Error())
		}

		var calibratedAmount sdk.Int
		if tokenInfo.ContractDecimal >= cmmtypes.TokenDecimals {
			decimals := sdk.NewIntWithDecimal(1, int(tokenInfo.ContractDecimal-cmmtypes.TokenDecimals))
			calibratedAmount = sdk.NewInt(transferClaim.Amount.Amount).Mul(decimals)
		} else {
			decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-tokenInfo.ContractDecimal))
			if !sdk.NewInt(transferClaim.Amount.Amount).Mod(decimals).IsZero() {
				return oracle.Prophecy{}, types.ErrInvalidAmount("can't calibrate timeout amount")
			}
			calibratedAmount = sdk.NewInt(transferClaim.Amount.Amount).Div(decimals)
		}

		if transferClaim.ExpireTime < ctx.BlockHeader().Time.Unix() {
			timeOutPackage, err := types.SerializeTimeoutPackage(calibratedAmount,
				transferClaim.ContractAddress[:], transferClaim.SenderAddress[:])

			if err != nil {
				return oracle.Prophecy{}, types.ErrSerializePackageFailed(err.Error())
			}

			timeoutChannelId, err := sdk.GetChannelID(types.TimeoutChannelName)
			if err != nil {
				return oracle.Prophecy{}, types.ErrGetChannelIdFailed(err.Error())
			}

			sdkErr := k.IbcKeeper.CreateIBCPackage(ctx, sdk.CrossChainID(k.DestChainId), timeoutChannelId, timeOutPackage)
			if sdkErr != nil {
				return oracle.Prophecy{}, sdkErr
			}
			return prophecy, nil
		}

		_, err = k.BankKeeper.SendCoins(ctx, types.PegAccount, transferClaim.ReceiverAddress, sdk.Coins{transferClaim.Amount})
		if err != nil {
			return oracle.Prophecy{}, err
		}

		// TODO distribute delay fee

		// TODO should we delete prophecy when prophecy succeeds

		// increase sequence
		k.IncreaseSequence(ctx, types.KeyCurrentTransferInSequence)
	} else if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	}

	return prophecy, nil
}

func (k Keeper) ProcessTimeoutClaim(ctx sdk.Context, claim oracle.Claim) (oracle.Prophecy, sdk.Error) {
	prophecy, err := k.oracleKeeper.ProcessClaim(ctx, claim)
	if err != nil {
		return oracle.Prophecy{}, err
	}

	if prophecy.Status.Text == oracle.SuccessStatusText {
		timeoutClaim, err := types.GetTransferOutTimeoutClaimFromOracleClaim(prophecy.Status.FinalClaim)
		if err != nil {
			return oracle.Prophecy{}, err
		}

		_, err = k.BankKeeper.SendCoins(ctx, types.PegAccount, timeoutClaim.SenderAddress, sdk.Coins{timeoutClaim.Amount})
		if err != nil {
			return oracle.Prophecy{}, err
		}

		k.IncreaseSequence(ctx, types.KeyTransferOutTimeoutSequence)
	} else if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	}
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
			bindRequest.Amount != updateBindClaim.Amount ||
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
			_, err = k.BankKeeper.SendCoins(ctx, types.PegAccount, bindRequest.From,
				sdk.Coins{sdk.Coin{Denom: bindRequest.Symbol, Amount: bindRequest.Amount}})
			if err != nil {
				return oracle.Prophecy{}, err
			}
		}

		k.DeleteBindRequest(ctx, updateBindClaim.Symbol)

		// TODO Distribute fee
		k.IncreaseSequence(ctx, types.KeyTransferOutTimeoutSequence)
	} else if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	}
	return prophecy, nil
}

func (k Keeper) CreateBindRequest(ctx sdk.Context, req types.BindRequest) sdk.Error {
	key := types.GetBindRequestKey(req.Symbol)

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(key)
	if bz != nil {
		return types.ErrBindRequestExists(fmt.Sprintf("bind request of %s already exists", req.Symbol))
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return sdk.ErrInternal(fmt.Sprintf("marshal bind request error, err=%s", err.Error()))
	}

	store.Set(key, reqBytes)
	return nil
}

func (k Keeper) DeleteBindRequest(ctx sdk.Context, symbol string) {
	key := types.GetBindRequestKey(symbol)

	store := ctx.KVStore(k.storeKey)
	store.Delete(key)
}

func (k Keeper) GetBindRequest(ctx sdk.Context, symbol string) (types.BindRequest, sdk.Error) {
	key := types.GetBindRequestKey(symbol)

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(key)
	if bz == nil {
		return types.BindRequest{}, types.ErrBindRequestNotExists(fmt.Sprintf("bind request of %s does not exist", symbol))
	}

	var bindRequest types.BindRequest
	err := json.Unmarshal(bz, &bindRequest)
	if err != nil {
		return types.BindRequest{}, sdk.ErrInternal(fmt.Sprintf("unmarshal bind request error, err=%s", err.Error()))
	}

	return bindRequest, nil
}
