package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/tendermint/tendermint/crypto"

	"github.com/binance-chain/node/plugins/bridge/types"
	"github.com/binance-chain/node/plugins/oracle"
)

var (
	// bnb prefix address:  bnb1v8vkkymvhe2sf7gd2092ujc6hweta38xadu2pj
	// tbnb prefix address: tbnb1v8vkkymvhe2sf7gd2092ujc6hweta38xnc4wpr
	PegAccount = sdk.AccAddress(crypto.AddressHash([]byte("BinanceChainPegAccount")))
)

// Keeper maintains the link to data storage and
// exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc *codec.Codec // The wire codec for binary encoding/decoding.

	oracleKeeper oracle.Keeper

	storeKey sdk.StoreKey // The key used to access the store from the Context.

	// The reference to the CoinKeeper to modify balances
	bankKeeper bank.Keeper
}

// NewKeeper creates new instances of the bridge Keeper
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, oracleKeeper oracle.Keeper, bankKeeper bank.Keeper) Keeper {
	return Keeper{
		cdc:          cdc,
		storeKey:     storeKey,
		bankKeeper:   bankKeeper,
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

		_, err = k.bankKeeper.SendCoins(ctx, PegAccount, transferClaim.ReceiverAddress, sdk.Coins{transferClaim.Amount})
		if err != nil {
			return oracle.Prophecy{}, err
		}

		// TODO distribute delay fee

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

		_, err = k.bankKeeper.SendCoins(ctx, timeoutClaim.SenderAddress, PegAccount, sdk.Coins{timeoutClaim.Amount})
		if err != nil {
			return oracle.Prophecy{}, err
		}

		k.IncreaseSequence(ctx, types.KeyTimeoutSequence)
	} else if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	}
	return prophecy, nil
}
