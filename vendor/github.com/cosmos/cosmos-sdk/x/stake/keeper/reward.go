package keeper

import (
	"encoding/binary"
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

const threshold = 5

func allocate(sharers []types.Sharer, totalRewards sdk.Dec) (rewards []types.PreReward) {
	var minToDistribute int64
	var shouldCarry []types.PreReward
	var shouldNotCarry []types.PreReward

	totalShares := sdk.ZeroDec()
	for _, sharer := range sharers {
		totalShares = totalShares.Add(sharer.Shares)
	}

	for _, sharer := range sharers {

		afterRoundDown, firstDecimalValue := mulQuoDecWithExtraDecimal(sharer.Shares, totalRewards, totalShares, 1)

		if firstDecimalValue < threshold {
			shouldNotCarry = append(shouldNotCarry, types.PreReward{AccAddr: sharer.AccAddr, Shares: sharer.Shares, Amount: afterRoundDown})
		} else {
			shouldCarry = append(shouldCarry, types.PreReward{AccAddr: sharer.AccAddr, Shares: sharer.Shares, Amount: afterRoundDown})
		}
		minToDistribute += afterRoundDown
	}
	remainingRewards := totalRewards.RawInt() - minToDistribute
	if remainingRewards > 0 {
		for i := range shouldCarry {
			if remainingRewards <= 0 {
				break
			} else {
				shouldCarry[i].Amount++
				remainingRewards--
			}
		}
		if remainingRewards > 0 {
			for i := range shouldNotCarry {
				if remainingRewards <= 0 {
					break
				} else {
					shouldNotCarry[i].Amount++
					remainingRewards--
				}
			}
		}
	}

	return append(shouldCarry, shouldNotCarry...)
}

// calculate a * b / c, getting the extra decimal digital as result of extraDecimalValue. For example:
// 0.00000003 * 2 / 0.00000004 = 0.000000015,
// assume that decimal place number of Dec is 8, and the extraDecimalPlace was given 1, then
// we take the 8th decimal place value '1' as afterRoundDown, and extra decimal value(9th) '5' as extraDecimalValue
func mulQuoDecWithExtraDecimal(a, b, c sdk.Dec, extraDecimalPlace int) (afterRoundDown int64, extraDecimalValue int) {
	extra := int64(math.Pow(10, float64(extraDecimalPlace)))
	product, ok := sdk.Mul64(a.RawInt(), b.RawInt())
	if !ok { // int64 exceed
		return mulQuoBigIntWithExtraDecimal(big.NewInt(a.RawInt()), big.NewInt(b.RawInt()), big.NewInt(c.RawInt()), big.NewInt(extra))
	} else {
		if product, ok = sdk.Mul64(product, extra); !ok {
			return mulQuoBigIntWithExtraDecimal(big.NewInt(a.RawInt()), big.NewInt(b.RawInt()), big.NewInt(c.RawInt()), big.NewInt(extra))
		}
		resultOfAddDecimalPlace := product / c.RawInt()
		afterRoundDown = resultOfAddDecimalPlace / extra
		extraDecimalValue = int(resultOfAddDecimalPlace % extra)
		return afterRoundDown, extraDecimalValue
	}
}

func mulQuoBigIntWithExtraDecimal(a, b, c, extra *big.Int) (afterRoundDown int64, extraDecimalValue int) {
	product := sdk.MulBigInt(sdk.MulBigInt(a, b), extra)
	result := sdk.QuoBigInt(product, c)

	expectedDecimalValueBig := &big.Int{}
	afterRoundDownBig, expectedDecimalValueBig := result.QuoRem(result, extra, expectedDecimalValueBig)
	afterRoundDown = afterRoundDownBig.Int64()
	extraDecimalValue = int(expectedDecimalValueBig.Int64())
	return afterRoundDown, extraDecimalValue
}

//___________________________________________________________________________

func getRewardBatchKey(batchNo int64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(batchNo))
	return append(RewardBatchKey, bz...)
}

func (k Keeper) hasNextBatchRewards(ctx sdk.Context) bool {
	store := ctx.KVStore(k.rewardStoreKey)

	iterator := sdk.KVStorePrefixIterator(store, RewardBatchKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		return true
	}
	return false
}

func (k Keeper) countBatchRewards(ctx sdk.Context) (count int64) {
	store := ctx.KVStore(k.rewardStoreKey)

	iterator := sdk.KVStorePrefixIterator(store, RewardBatchKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		count = count + 1
	}
	return
}

func (k Keeper) getNextBatchRewards(ctx sdk.Context) (rewards []types.Reward, key []byte) {
	store := ctx.KVStore(k.rewardStoreKey)

	iterator := sdk.KVStorePrefixIterator(store, RewardBatchKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		value := iterator.Value()
		rewards = types.MustUnmarshalRewards(k.cdc, value)
		key = iterator.Key()
		return
	}
	return nil, nil
}

func (k Keeper) setBatchRewards(ctx sdk.Context, batchNo int64, rewards []types.Reward) {
	store := ctx.KVStore(k.rewardStoreKey)
	bz := types.MustMarshalRewards(k.cdc, rewards)
	store.Set(getRewardBatchKey(batchNo), bz)
}

func (k Keeper) removeBatchRewards(ctx sdk.Context, key []byte) {
	store := ctx.KVStore(k.rewardStoreKey)
	store.Delete(key)
}

func (k Keeper) setRewardValDistAddrs(ctx sdk.Context, valDistAddrs []types.StoredValDistAddr) {
	store := ctx.KVStore(k.rewardStoreKey)
	bz := types.MustMarshalValDistAddrs(k.cdc, valDistAddrs)
	store.Set(RewardValDistAddrKey, bz)
}

func (k Keeper) getRewardValDistAddrs(ctx sdk.Context) (valDistAddrs []types.StoredValDistAddr, found bool) {
	store := ctx.KVStore(k.rewardStoreKey)
	value := store.Get(RewardValDistAddrKey)
	if value != nil {
		return types.MustUnmarshalValDistAddrs(k.cdc, value), true
	}
	return nil, false
}

func (k Keeper) removeRewardValDistAddrs(ctx sdk.Context) {
	store := ctx.KVStore(k.rewardStoreKey)
	store.Delete(RewardValDistAddrKey)
}
