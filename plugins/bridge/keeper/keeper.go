package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

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

	// The reference to the CoinKeeper to modify balances
	BankKeeper bank.Keeper

	TokenMapper store.Mapper
}

// NewKeeper creates new instances of the bridge Keeper
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, tokenMapper store.Mapper, oracleKeeper oracle.Keeper, bankKeeper bank.Keeper) Keeper {
	return Keeper{
		cdc:          cdc,
		storeKey:     storeKey,
		BankKeeper:   bankKeeper,
		TokenMapper:  tokenMapper,
		oracleKeeper: oracleKeeper,
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

		if transferClaim.ExpireTime < ctx.BlockHeader().Time.Unix() {
			// TODO write timeout package when timeout
			return oracle.Prophecy{}, types.ErrInvalidExpireTime(fmt.Sprintf("expire time(%d) is before now(%d)",
				transferClaim.ExpireTime, ctx.BlockHeader().Time.Unix()))
		}

		_, err = k.BankKeeper.SendCoins(ctx, types.PegAccount, transferClaim.ReceiverAddress, sdk.Coins{transferClaim.Amount})
		if err != nil {
			return oracle.Prophecy{}, err
		}

		// TODO distribute delay fee

		// TODO should we delete prophecy when prophecy succeeds

		// increase sequence
		k.IncreaseSequence(ctx, types.KeyCurrentTransferSequence)
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
		timeoutClaim, err := types.GetTimeoutClaimFromOracleClaim(prophecy.Status.FinalClaim)
		if err != nil {
			return oracle.Prophecy{}, err
		}

		_, err = k.BankKeeper.SendCoins(ctx, timeoutClaim.SenderAddress, types.PegAccount, sdk.Coins{timeoutClaim.Amount})
		if err != nil {
			return oracle.Prophecy{}, err
		}

		k.IncreaseSequence(ctx, types.KeyTimeoutSequence)
	} else if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	}
	return prophecy, nil
}
