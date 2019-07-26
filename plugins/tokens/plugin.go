package tokens

import (
	"encoding/binary"
	"fmt"
	"time"

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
	timeLockKeeper timelock.Keeper) {
	// add msg handlers
	for route, handler := range Routes(mapper, accKeeper, coinKeeper, timeLockKeeper) {
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
func EndBreatheBlock(ctx sdk.Context, swapKeeper swap.Keeper, height int64, blockTime time.Time) {
	logger := bnclog.With("module", "tokens")

	logger.Info("Delete swaps which are completed or expired", "blockHeight", height)

	iterator := swapKeeper.GetSwapTimerIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		if len(key) != 1+8+swap.RandomNumberHashLength {
			logger.Error("Unexpected key length", "expectedLength", 1+8+swap.RandomNumberHashLength, "actualLength", len(key))
			continue
		}
		swapClosedTime := int64(binary.BigEndian.Uint64(key[1:1+8]))
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
		err := swapKeeper.DeleteSwap(ctx, swapRecord.RandomNumberHash)
		if err != nil {
			logger.Error(fmt.Sprintf("Encounter error in deleting swaps which were completed or expired: %s", err.Error()))
			continue
		}
	}

}
