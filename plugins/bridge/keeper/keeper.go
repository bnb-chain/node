package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/tendermint/tendermint/crypto"

	"github.com/binance-chain/node/plugins/bridge/types"
	"github.com/binance-chain/node/plugins/oracle"
)

var (
	PegAccount = sdk.AccAddress(crypto.AddressHash([]byte("BinanceChainPegAccount")))
)

// Keeper maintains the link to data storage and
// exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc *codec.Codec // The wire codec for binary encoding/decoding.

	oracleKeeper oracle.Keeper

	// The reference to the CoinKeeper to modify balances
	bankKeeper bank.Keeper
}

// NewKeeper creates new instances of the bridge Keeper
func NewKeeper(cdc *codec.Codec, oracleKeeper oracle.Keeper, bankKeeper bank.Keeper) Keeper {
	return Keeper{
		cdc:          cdc,
		bankKeeper:   bankKeeper,
		oracleKeeper: oracleKeeper,
	}
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
	} else if prophecy.Status.Text == oracle.FailedStatusText {
		k.oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	}
	return prophecy, nil
}
