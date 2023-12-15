package tokens

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	bnclog "github.com/bnb-chain/node/common/log"
	"github.com/bnb-chain/node/common/types"
	app "github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/common/upgrade"
	"github.com/bnb-chain/node/plugins/tokens/swap"
	"github.com/bnb-chain/node/plugins/tokens/timelock"
)

const abciQueryPrefix = "tokens"
const miniAbciQueryPrefix = "mini-tokens"

// InitPlugin initializes the plugin.
func InitPlugin(
	appp app.ChainApp, mapper Mapper, accKeeper auth.AccountKeeper, coinKeeper bank.Keeper,
	timeLockKeeper timelock.Keeper, swapKeeper swap.Keeper) {
	// add msg handlers
	for route, handler := range Routes(mapper, accKeeper, coinKeeper, timeLockKeeper,
		swapKeeper) {
		appp.GetRouter().AddRoute(route, handler)
	}

	// add abci handlers
	tokenHandler := createQueryHandler(mapper, abciQueryPrefix)
	miniTokenHandler := createQueryHandler(mapper, miniAbciQueryPrefix)
	appp.RegisterQueryHandler(abciQueryPrefix, tokenHandler)
	appp.RegisterQueryHandler(miniAbciQueryPrefix, miniTokenHandler)
	RegisterUpgradeBeginBlocker(mapper)
}

func RegisterUpgradeBeginBlocker(mapper Mapper) {
	// bind bnb smart chain contract address to bnb token
	upgrade.Mgr.RegisterBeginBlocker(upgrade.LaunchBscUpgrade, func(ctx sdk.Context) {
		err := mapper.UpdateBind(ctx, types.NativeTokenSymbol, "0x0000000000000000000000000000000000000000", 18)
		if err != nil {
			panic(err)
		}
	})
}

func createQueryHandler(mapper Mapper, queryPrefix string) app.AbciQueryHandler {
	return createAbciQueryHandler(mapper, queryPrefix)
}

const (
	MaxUnlockItems = 10
)

func EndBlocker(ctx sdk.Context, timelockKeeper timelock.Keeper, swapKeeper swap.Keeper) {
	if !sdk.IsUpgrade(sdk.SecondSunsetFork) {
		return
	}
	logger := bnclog.With("module", "tokens")
	logger.Info("unlock the time locks", "blockHeight", ctx.BlockHeight())

	iterator := timelockKeeper.GetTimeLockRecordIterator(ctx)
	defer iterator.Close()
	i := 0
	for ; iterator.Valid(); iterator.Next() {
		if i >= MaxUnlockItems {
			break
		}
		addr, id, err := timelock.ParseKeyRecord(iterator.Key())
		if err != nil {
			logger.Error("ParseKeyRecord error", "error", err)
			continue
		}
		err = timelockKeeper.TimeUnlock(ctx, addr, id)
		if err != nil {
			logger.Error("TimeUnlock error", "error", err)
			continue
		}
		i++
	}

	swapIterator := swapKeeper.GetSwapIterator(ctx)
	defer swapIterator.Close()
	i = 0
	for ; swapIterator.Valid(); swapIterator.Next() {
		if i >= MaxUnlockItems {
			break
		}
		var automaticSwap swap.AtomicSwap
		swapKeeper.CDC().MustUnmarshalBinaryBare(swapIterator.Value(), &automaticSwap)
		swapID := swapIterator.Key()[len(swap.HashKey):]
		swapItem := swapKeeper.GetSwap(ctx, swapID)
		if swapItem == nil {
			continue
		}
		if swapItem.Status != swap.Open {
			continue
		}
		result := swap.HandleRefundHashTimerLockedTransferAfterBCFusion(ctx, swapKeeper, swap.RefundHTLTMsg{
			From:   automaticSwap.From,
			SwapID: swapID,
		})
		if !result.IsOK() {
			logger.Error("Refund error", "swapId", swapID)
			continue
		}
		i++
	}
}

// EndBreatheBlock processes the breathe block lifecycle event.
func EndBreatheBlock(ctx sdk.Context, swapKeeper swap.Keeper) {
	logger := bnclog.With("module", "tokens")

	logger.Info("Delete swaps which are completed or expired", "blockHeight", ctx.BlockHeight())

	iterator := swapKeeper.GetSwapCloseTimeIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		if len(key) != 1+swap.Int64Size+swap.Int64Size {
			swapKeeper.DeleteKey(ctx, key)
			logger.Error("Unexpected key length", "expectedLength", 1+swap.Int64Size+swap.Int64Size, "actualLength", len(key))
			continue
		}
		swapClosedTime := int64(binary.BigEndian.Uint64(key[1 : 1+swap.Int64Size]))
		// Only delete swaps which were closed one week ago
		if swapClosedTime > ctx.BlockHeader().Time.Unix()-swap.OneWeek {
			break
		}
		swapID := iterator.Value()
		swapRecord := swapKeeper.GetSwap(ctx, swapID)
		if swapRecord == nil {
			swapKeeper.DeleteKey(ctx, key)
			logger.Error("No matched swap", "swapID", swapID)
			continue
		}
		if swapRecord.Status != swap.Completed && swapRecord.Status != swap.Expired {
			logger.Error("Swap status must be completed or expired", "swapStatus", swapRecord.Status)
			continue
		}
		err := swapKeeper.DeleteSwap(ctx, swapID, swapRecord)
		if err != nil {
			logger.Error(fmt.Sprintf("Encounter error in deleting swaps which were completed or expired: %s", err.Error()))
			continue
		}
	}
}
