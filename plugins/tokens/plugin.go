package tokens

import (
	"encoding/binary"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	bnclog "github.com/binance-chain/node/common/log"
	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/plugins/tokens/timelock"
)

const abciQueryPrefix = "tokens"

// InitPlugin initializes the plugin.
func InitPlugin(
	appp app.ChainApp, mapper Mapper, accKeeper auth.AccountKeeper, coinKeeper bank.Keeper,
	timeLockKeeper timelock.Keeper, swapKeeper swap.Keeper) {
	// add msg handlers
	for route, handler := range Routes(mapper, accKeeper, coinKeeper, timeLockKeeper, swapKeeper) {
		appp.GetRouter().AddRoute(route, handler)
	}

	// add abci handlers
	handler := createQueryHandler(mapper)
	appp.RegisterQueryHandler(abciQueryPrefix, handler)
}

func createQueryHandler(mapper Mapper) app.AbciQueryHandler {
	return createAbciQueryHandler(mapper)
}

// EndBreatheBlock processes the breathe block lifecycle event.
func EndBreatheBlock(ctx sdk.Context, swapKeeper swap.Keeper) {
	logger := bnclog.With("module", "tokens")

	logger.Info("Delete swaps which are completed or expired", "blockHeight", ctx.BlockHeight())

	iterator := swapKeeper.GetSwapTimerIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		if len(key) != 1+swap.Int64Size+swap.Int64Size {
			logger.Error("Unexpected key length", "expectedLength", 1+swap.Int64Size+swap.Int64Size, "actualLength", len(key))
			continue
		}
		swapClosedTime := int64(binary.BigEndian.Uint64(key[1:1+swap.Int64Size]))
		// Only delete swaps which were closed one week ago
		if swapClosedTime > ctx.BlockHeader().Time.Unix() - 86400 * 7 {
			break
		}
		randomNumberHash := iterator.Value()
		swapRecord := swapKeeper.QuerySwap(ctx, randomNumberHash)
		if swapRecord == nil {
			logger.Error("Unexpected randomNumberHash, no corresponding swap record", "randomNumberHash", randomNumberHash)
			continue
		}
		if swapRecord.Status != swap.Completed && swapRecord.Status != swap.Expired {
			logger.Error("Swap status should be completed or expired", "swapStatus", swapRecord.Status)
			continue
		}
		err := swapKeeper.DeleteSwap(ctx, swapRecord)
		if err != nil {
			logger.Error(fmt.Sprintf("Encounter error in deleting swaps which were completed or expired: %s", err.Error()))
			continue
		}
	}

}
