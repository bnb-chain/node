package order

import (
	"errors"
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/param/paramhub"
	paramTypes "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"
)

type GlobalKeeper struct {
	am                         auth.AccountKeeper
	FeeManager                 *FeeManager
	RoundOrderFees             FeeHolder // order (and trade) related fee of this round, str of addr bytes -> fee
	CollectOrderInfoForPublish bool
	logger                     tmlog.Logger
}

func NewGlobalKeeper(cdc *wire.Codec, am auth.AccountKeeper, collectOrderInfoForPublish bool) *GlobalKeeper {
	logger := bnclog.With("module", "dex_global_keeper")
	return &GlobalKeeper{
		am:                         am,
		RoundOrderFees:             make(map[string]*types.Fee, 256),
		FeeManager:                 NewFeeManager(cdc, logger),
		CollectOrderInfoForPublish: collectOrderInfoForPublish,
		logger:                     logger,
	}
}

// deliberately make `fee` parameter not a pointer
// in case we modify the original fee (which will be referenced when distribute to validator)
func (kp *GlobalKeeper) updateRoundOrderFee(addr string, fee types.Fee) {
	if existingFee, ok := kp.RoundOrderFees[addr]; ok {
		existingFee.AddFee(fee)
	} else {
		kp.RoundOrderFees[addr] = &fee
	}
}

func (kp *GlobalKeeper) ClearRoundFee() {
	kp.RoundOrderFees = make(map[string]*types.Fee, 256)
}

func (kp *GlobalKeeper) allocate(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer), engines map[string]*matcheng.MatchEng) (
	types.Fee, map[string]*types.Fee) {
	if !sdk.IsUpgrade(upgrade.BEP19) {
		return kp.allocateBeforeGalileo(ctx, tranCh, postAllocateHandler, engines)
	}

	// use string of the addr as the key since map makes a fast path for string key.
	// Also, making the key have same length is also an optimization.
	tradeTransfers := make(map[string]TradeTransfers)
	// expire fee is fixed, so we count by numbers.
	expireTransfers := make(map[string]ExpireTransfers)
	// we need to distinguish different expire event, IOCExpire or Expire. only one of the two will exist.
	var expireEventType transferEventType
	var totalFee types.Fee
	for tran := range tranCh {
		kp.doTransfer(ctx, &tran)
		if !tran.FeeFree() {
			addrStr := string(tran.accAddress.Bytes())
			// need a copy of tran as it is reused
			tranCp := tran
			if tran.IsExpiredWithFee() {
				expireEventType = tran.eventType
				if _, ok := expireTransfers[addrStr]; !ok {
					expireTransfers[addrStr] = ExpireTransfers{&tranCp}
				} else {
					expireTransfers[addrStr] = append(expireTransfers[addrStr], &tranCp)
				}
			} else if tran.eventType == eventFilled {
				if _, ok := tradeTransfers[addrStr]; !ok {
					tradeTransfers[addrStr] = TradeTransfers{&tranCp}
				} else {
					tradeTransfers[addrStr] = append(tradeTransfers[addrStr], &tranCp)
				}
			}
		} else if tran.IsExpire() {
			if postAllocateHandler != nil {
				postAllocateHandler(tran)
			}
		}
	}

	feesPerAcc := make(map[string]*types.Fee)
	for addrStr, trans := range tradeTransfers {
		addr := sdk.AccAddress(addrStr)
		acc := kp.am.GetAccount(ctx, addr)
		fees := kp.FeeManager.CalcTradesFee(acc.GetCoins(), trans, engines)
		if !fees.IsEmpty() {
			feesPerAcc[addrStr] = &fees
			acc.SetCoins(acc.GetCoins().Minus(fees.Tokens))
			kp.am.SetAccount(ctx, acc)
			totalFee.AddFee(fees)
		}
	}

	for addrStr, trans := range expireTransfers {
		addr := sdk.AccAddress(addrStr)
		acc := kp.am.GetAccount(ctx, addr)

		fees := kp.FeeManager.CalcExpiresFee(acc.GetCoins(), expireEventType, trans, engines, postAllocateHandler)
		if !fees.IsEmpty() {
			if _, ok := feesPerAcc[addrStr]; ok {
				feesPerAcc[addrStr].AddFee(fees)
			} else {
				feesPerAcc[addrStr] = &fees
			}
			acc.SetCoins(acc.GetCoins().Minus(fees.Tokens))
			kp.am.SetAccount(ctx, acc)
			totalFee.AddFee(fees)
		}
	}
	return totalFee, feesPerAcc
}

func (kp *GlobalKeeper) allocateAndCalcFee(
	ctx sdk.Context,
	tradeOuts []chan Transfer,
	postAlloTransHandler TransferHandler,
	engines map[string]*matcheng.MatchEng) types.Fee {
	concurrency := len(tradeOuts)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	feesPerCh := make([]types.Fee, concurrency)
	feesPerAcc := make([]map[string]*types.Fee, concurrency)
	allocatePerCh := func(index int, tranCh <-chan Transfer) {
		defer wg.Done()
		fee, feeByAcc := kp.allocate(ctx, tranCh, postAlloTransHandler, engines)
		feesPerCh[index].AddFee(fee)
		feesPerAcc[index] = feeByAcc
	}

	for i, tradeTranCh := range tradeOuts {
		go allocatePerCh(i, tradeTranCh)
	}
	wg.Wait()
	totalFee := types.Fee{}
	for i := 0; i < concurrency; i++ {
		totalFee.AddFee(feesPerCh[i])
	}
	if kp.CollectOrderInfoForPublish {
		for _, m := range feesPerAcc {
			for k, v := range m {
				kp.updateRoundOrderFee(k, *v)
			}
		}
	}
	return totalFee
}

// DEPRECATED
func (kp *GlobalKeeper) allocateBeforeGalileo(ctx sdk.Context, tranCh <-chan Transfer, postAllocateHandler func(tran Transfer), engines map[string]*matcheng.MatchEng) (
	types.Fee, map[string]*types.Fee) {
	// use string of the addr as the key since map makes a fast path for string key.
	// Also, making the key have same length is also an optimization.
	tradeInAsset := make(map[string]*sortedAsset)
	// expire fee is fixed, so we count by numbers.
	expireInAsset := make(map[string]*sortedAsset)
	// we need to distinguish different expire event, IOCExpire or Expire. only one of the two will exist.
	var expireEventType transferEventType
	var totalFee types.Fee
	for tran := range tranCh {
		kp.doTransfer(ctx, &tran)
		if !tran.FeeFree() {
			addrStr := string(tran.accAddress.Bytes())
			if tran.IsExpiredWithFee() {
				expireEventType = tran.eventType
				fees, ok := expireInAsset[addrStr]
				if !ok {
					fees = &sortedAsset{}
					expireInAsset[addrStr] = fees
				}
				fees.addAsset(tran.inAsset, 1)
			} else if tran.eventType == eventFilled {
				fees, ok := tradeInAsset[addrStr]
				if !ok {
					fees = &sortedAsset{}
					tradeInAsset[addrStr] = fees
				}
				// no possible to overflow, for tran.in == otherSide.tran.out <= TotalSupply(otherSide.tran.outAsset)
				fees.addAsset(tran.inAsset, tran.in)
			}
		}
		if postAllocateHandler != nil {
			postAllocateHandler(tran)
		}
	}

	feesPerAcc := make(map[string]*types.Fee)
	collectFee := func(assetsMap map[string]*sortedAsset, calcFeeAndDeduct func(acc sdk.Account, in sdk.Coin) types.Fee) {
		for addrStr, assets := range assetsMap {
			addr := sdk.AccAddress(addrStr)
			acc := kp.am.GetAccount(ctx, addr)

			var fees types.Fee
			if exists, ok := feesPerAcc[addrStr]; ok {
				fees = *exists
			}
			if assets.native != 0 {
				fee := calcFeeAndDeduct(acc, sdk.NewCoin(types.NativeTokenSymbol, assets.native))
				fees.AddFee(fee)
				totalFee.AddFee(fee)
			}
			for _, asset := range assets.tokens {
				fee := calcFeeAndDeduct(acc, asset)
				fees.AddFee(fee)
				totalFee.AddFee(fee)
			}
			if !fees.IsEmpty() {
				feesPerAcc[addrStr] = &fees
				kp.am.SetAccount(ctx, acc)
			}
		}
	}
	collectFee(tradeInAsset, func(acc sdk.Account, in sdk.Coin) types.Fee {
		fee := kp.FeeManager.CalcTradeFee(acc.GetCoins(), in, engines)
		acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
		return fee
	})
	collectFee(expireInAsset, func(acc sdk.Account, in sdk.Coin) types.Fee {
		var i int64 = 0
		var fees types.Fee
		for ; i < in.Amount; i++ {
			fee := kp.FeeManager.CalcFixedFee(acc.GetCoins(), expireEventType, in.Denom, engines)
			acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
			fees.AddFee(fee)
		}
		return fees
	})
	return totalFee, feesPerAcc
}

func (kp *GlobalKeeper) doTransfer(ctx sdk.Context, tran *Transfer) sdk.Error {
	account := kp.am.GetAccount(ctx, tran.accAddress).(types.NamedAccount)
	newLocked := account.GetLockedCoins().Minus(sdk.Coins{sdk.NewCoin(tran.outAsset, tran.unlock)})
	// these two non-negative check are to ensure the Transfer gen result is correct before we actually operate the acc.
	// they should never happen, there would be a severe bug if happen and we have to cancel all orders when app restarts.
	if !newLocked.IsNotNegative() {
		panic(fmt.Errorf(
			"no enough locked tokens to unlock, oid: %s, newLocked: %s, unlock: %d",
			tran.Oid,
			newLocked.String(),
			tran.unlock))
	}
	if tran.unlock < tran.out {
		panic(errors.New("unlocked tokens cannot cover the expense"))
	}
	account.SetLockedCoins(newLocked)
	accountCoin := account.GetCoins().
		Plus(sdk.Coins{sdk.NewCoin(tran.inAsset, tran.in)})
	if remain := tran.unlock - tran.out; remain > 0 || !sdk.IsUpgrade(upgrade.FixZeroBalance) {
		accountCoin = accountCoin.Plus(sdk.Coins{sdk.NewCoin(tran.outAsset, remain)})
	}
	account.SetCoins(accountCoin)

	kp.am.SetAccount(ctx, account)
	return nil
}

func (kp *GlobalKeeper) SubscribeParamChange(hub *paramhub.Keeper) {
	hub.SubscribeParamChange(
		func(ctx sdk.Context, changes []interface{}) {
			for _, c := range changes {
				switch change := c.(type) {
				case []paramTypes.FeeParam:
					feeConfig := ParamToFeeConfig(change)
					if feeConfig != nil {
						kp.FeeManager.UpdateConfig(*feeConfig)
					}
				default:
					kp.logger.Debug("Receive param changes that not interested.")
				}
			}
		},
		func(context sdk.Context, state paramTypes.GenesisState) {
			feeConfig := ParamToFeeConfig(state.FeeGenesis)
			if feeConfig != nil {
				kp.FeeManager.UpdateConfig(*feeConfig)
			} else {
				panic("Genesis with no dex fee config ")
			}
		},
		func(context sdk.Context, iLoad interface{}) {
			switch load := iLoad.(type) {
			case []paramTypes.FeeParam:
				feeConfig := ParamToFeeConfig(load)
				if feeConfig != nil {
					kp.FeeManager.UpdateConfig(*feeConfig)
				} else {
					panic("Load with no dex fee config ")
				}
			default:
				kp.logger.Debug("Receive param load that not interested.")
			}
		})
}
