package pub

import (
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/types"
	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

func GetAccountBalances(mapper auth.AccountMapper, ctx sdk.Context, accSlices ...[]string) (res map[string]Account) {
	res = make(map[string]Account)

	for _, accs := range accSlices {
		for _, addrBytesStr := range accs {
			if _, ok := res[addrBytesStr]; !ok {
				addr := sdk.AccAddress([]byte(addrBytesStr))
				if acc, ok := mapper.GetAccount(ctx, addr).(types.NamedAccount); ok {
					assetsMap := make(map[string]*AssetBalance)
					// TODO(#66): set the length to be the total coins this account owned
					assets := make([]AssetBalance, 0, 10)

					for _, freeCoin := range acc.GetCoins() {
						if assetBalance, ok := assetsMap[freeCoin.Denom]; ok {
							assetBalance.Free = freeCoin.Amount.Int64()
						} else {
							newAB := AssetBalance{Asset: freeCoin.Denom, Free: freeCoin.Amount.Int64()}
							assets = append(assets, newAB)
							assetsMap[freeCoin.Denom] = &newAB
						}
					}

					for _, frozenCoin := range acc.GetFrozenCoins() {
						if assetBalance, ok := assetsMap[frozenCoin.Denom]; ok {
							assetBalance.Frozen = frozenCoin.Amount.Int64()
						} else {
							newAB := AssetBalance{Asset: frozenCoin.Denom, Frozen: frozenCoin.Amount.Int64()}
							assets = append(assets, newAB)
							assetsMap[frozenCoin.Denom] = &newAB
						}
					}

					for _, lockedCoin := range acc.GetLockedCoins() {
						if assetBalance, ok := assetsMap[lockedCoin.Denom]; ok {
							assetBalance.Locked = lockedCoin.Amount.Int64()
						} else {
							newAB := AssetBalance{Asset: lockedCoin.Denom, Locked: lockedCoin.Amount.Int64()}
							assets = append(assets, newAB)
							assetsMap[lockedCoin.Denom] = &newAB
						}
					}

					bech32Str := addr.String()
					res[bech32Str] = Account{bech32Str, assets}
				} else {
					Logger.Error(fmt.Sprintf("failed to get account %s from AccountMapper", addr.String()))
				}
			}
		}
	}

	return
}

func MatchAndAllocateAllForPublish(
	dexKeeper *orderPkg.Keeper,
	accountMapper auth.AccountMapper,
	ctx sdk.Context) []Trade {
	tradeFeeHolderCh := make(chan orderPkg.TradeFeeHolder, FeeCollectionChannelSize)
	iocExpireFeeHolderCh := make(chan orderPkg.ExpireFeeHolder, FeeCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(2)

	// group trades by Bid and Sid to make fee update easier
	tradesToPublish := make([]Trade, 0)
	go collectTradeFeeForPublish(&tradesToPublish, &wg, ctx.BlockHeader().Height, tradeFeeHolderCh)
	go updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh, orderPkg.IocNoFill)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		var fee orderPkg.Fee
		if !tran.Fee.IsEmpty() {
			// TODO(#160): Fix potential fee precision loss
			fee = orderPkg.Fee{tran.Fee.Tokens[0].Amount.Int64(), tran.Fee.Tokens[0].Denom}
		}

		if tran.IsExpire() {
			iocExpireFeeHolderCh <- orderPkg.ExpireFeeHolder{tran.Oid, fee}
		} else {
			tradeFeeHolderCh <- orderPkg.TradeFeeHolder{tran.Oid, tran.Trade, tran.Symbol, fee}
		}
	}
	ctx, _, _ = dexKeeper.MatchAndAllocateAll(ctx, accountMapper, feeCollectorForTrades)
	close(tradeFeeHolderCh)
	close(iocExpireFeeHolderCh)
	wg.Wait()

	return tradesToPublish
}

func ExpireOrdersForPublish(
	dexKeeper *orderPkg.Keeper,
	accountMapper auth.AccountMapper,
	ctx sdk.Context,
	blockTime int64) {
	expireFeeHolderCh := make(chan orderPkg.ExpireFeeHolder, FeeCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go updateExpireFeeForPublish(dexKeeper, &wg, expireFeeHolderCh, orderPkg.Expired)
	var feeCollectorForTrades = func(tran orderPkg.Transfer) {
		var fee orderPkg.Fee
		if !tran.Fee.IsEmpty() {
			// TODO(#160): Fix potential fee precision loss
			fee = orderPkg.Fee{tran.Fee.Tokens[0].Amount.Int64(), tran.Fee.Tokens[0].Denom}
		}
		if tran.IsExpire() {
			expireFeeHolderCh <- orderPkg.ExpireFeeHolder{tran.Oid, fee}
		}
	}
	dexKeeper.ExpireOrders(ctx, blockTime, accountMapper, feeCollectorForTrades)
	close(expireFeeHolderCh)
	wg.Wait()
}

// for partial and fully filled order fee
func collectTradeFeeForPublish(
	tradesToPublish *[]Trade,
	wg *sync.WaitGroup,
	height int64,
	feeHolderCh <-chan orderPkg.TradeFeeHolder) {

	defer wg.Done()
	tradeIdx := 0
	trades := make(map[*me.Trade]*Trade)
	for feeHolder := range feeHolderCh {
		Logger.Debug("processing TradeFeeHolder", "feeHolder", feeHolder.String())
		var t *Trade
		// one trade has two transfer, the second fee update should applied to first updated trade
		if trade, ok := trades[feeHolder.Trade]; !ok {
			t = &Trade{
				Id:     fmt.Sprintf("%d-%d", height, tradeIdx),
				Symbol: feeHolder.Symbol,
				Sid:    feeHolder.Trade.Sid,
				Bid:    feeHolder.Trade.Bid,
				Price:  feeHolder.Trade.LastPx,
				Qty:    feeHolder.Trade.LastQty,
				Bfee:   -1,
				Sfee:   -1}
			trades[feeHolder.Trade] = t
			tradeIdx += 1
		} else {
			t = trade
		}

		if feeHolder.OId == feeHolder.Trade.Bid {
			t.Bfee = feeHolder.Amount
			t.BfeeAsset = feeHolder.Asset
		} else {
			t.Sfee = feeHolder.Amount
			t.SfeeAsset = feeHolder.Asset
		}

		if t.Bfee != -1 && t.Sfee != -1 {
			*tradesToPublish = append(*tradesToPublish, *t)
		}
	}
}

func updateExpireFeeForPublish(
	dexKeeper *orderPkg.Keeper,
	wg *sync.WaitGroup,
	feeHolderCh <-chan orderPkg.ExpireFeeHolder,
	reason orderPkg.ChangeType) {
	defer wg.Done()
	for feeHolder := range feeHolderCh {
		Logger.Debug("fee Collector for expire transfer", "transfer", feeHolder.String())

		id := feeHolder.OrderId
		originOrd := dexKeeper.OrderChangesMap[id]
		var fee int64
		var feeAsset string
		fee = feeHolder.Amount
		feeAsset = feeHolder.Asset
		change := orderPkg.OrderChange{originOrd.Id, reason, fee, feeAsset}
		dexKeeper.OrderChanges = append(dexKeeper.OrderChanges, change)
	}
}
