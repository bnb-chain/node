package pub

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/types"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/utils"
	miniIssue "github.com/binance-chain/node/plugins/minitokens/issue"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
	abci "github.com/tendermint/tendermint/abci/types"
)

func GetTradeAndOrdersRelatedAccounts(kp *orderPkg.DexKeeper, tradesToPublish []*Trade, pairType orderPkg.SymbolPairType) []string {
	res := make([]string, 0, len(tradesToPublish)*2+len(kp.GetOrderChanges(pairType)))
	OrderInfosForPub := kp.GetOrderInfosForPub(pairType)

	for _, t := range tradesToPublish {

		if bo, ok := OrderInfosForPub[t.Bid]; ok {
			res = append(res, string(bo.Sender.Bytes()))
		} else {
			Logger.Error("failed to locate buy order in OrderChangesMap for trade account resolving", "bid", t.Bid)
		}
		if so, ok := OrderInfosForPub[t.Sid]; ok {
			res = append(res, string(so.Sender.Bytes()))
		} else {
			Logger.Error("failed to locate sell order in OrderChangesMap for trade account resolving", "sid", t.Sid)
		}
	}

	for _, orderChange := range kp.GetOrderChanges(pairType) {
		if orderInfo := OrderInfosForPub[orderChange.Id]; orderInfo != nil {
			res = append(res, string(orderInfo.Sender.Bytes()))
		} else {
			Logger.Error("failed to locate order change in OrderChangesMap", "orderChange", orderChange.String())
		}
	}

	return res
}

func GetBlockPublished(pool *sdk.Pool, header abci.Header, blockHash []byte) *Block {
	txs := pool.GetTxs()
	transactionsToPublish := make([]Transaction, 0, 0)
	timeStamp := header.GetTime().Format(time.RFC3339Nano)
	txs.Range(func(key, value interface{}) bool {
		txhash := key.(string)
		stdTx, ok := value.(auth.StdTx)
		if !ok {
			Logger.Error("tx is not an auth.StdTx", "hash", txhash)
			return true
		}

		msgs := stdTx.GetMsgs()
		if len(msgs) == 0 {
			Logger.Error("tx contains no messages", "hash", txhash)
			return true
		}
		// TODO, if support multi message in one transaction in future, this part need change, refer issue #681
		msg := msgs[0]
		bz, err := json.Marshal(msg)
		if err != nil {
			Logger.Error("MarshalJSON message failed", "err", err)
			return true
		}
		fee := fees.Pool.GetFee(txhash)
		feeStr := ""
		if fee != nil {
			feeStr = fee.Tokens.String()
		}
		txRes := Pool.GetTxRes(txhash)
		if txRes == nil {
			Logger.Error("failed to get tx res", "hash", txhash)
			return true
		}
		var txAsset string
		var orderId string
		var outputs []Output
		var proposalId int64
		inputs := []Input{{Address: msg.GetSigners()[0].String()}}

		switch msg := msg.(type) {
		case orderPkg.NewOrderMsg:
			var orderRes orderPkg.NewOrderResponse
			err = json.Unmarshal([]byte(txRes.Data), &orderRes)
			if err != nil {
				Logger.Error("failed to get order id", "err", err)
				return true
			}
			orderId = orderRes.OrderID
			txAsset = msg.Symbol
		case gov.MsgSubmitProposal:
			proposalIdStr := string(txRes.Data)
			proposalId, err = strconv.ParseInt(proposalIdStr, 10, 64)
			if err != nil {
				Logger.Error("failed to parse proposalId")
				return true
			}
		case orderPkg.CancelOrderMsg:
			orderId = msg.RefId
			txAsset = msg.Symbol
		case bank.MsgSend:
			// TODO for now there is no requirement to support multi send message, will support multi send in issue #680
			txAsset = msg.Inputs[0].Coins[0].Denom
			inputs = transferInputsToPublish(msg.Inputs)
			outputs = transferOutputsToPublish(msg.Outputs)
		case gov.MsgDeposit:
			txAsset = types.NativeTokenSymbol
		case issue.IssueMsg:
			txAsset = msg.Symbol
		case issue.MintMsg:
			txAsset = msg.Symbol
		case burn.BurnMsg:
			txAsset = msg.Symbol
		case freeze.FreezeMsg:
			txAsset = msg.Symbol
		case freeze.UnfreezeMsg:
			txAsset = msg.Symbol
			// will not cover timelock, timeUnlock, timeRelock, atomic Swap
		case miniIssue.IssueMsg:
			txAsset = msg.Symbol
		}
		transactionsToPublish = append(transactionsToPublish, Transaction{
			TxHash:    txhash,
			Timestamp: timeStamp,
			Fee:       feeStr,
			Inputs:    inputs,
			Outputs:   outputs,
			NativeTransaction: NativeTransaction{
				Source:     stdTx.Source,
				ProposalId: proposalId,
				TxType:     msg.Type(),
				TxAsset:    txAsset,
				OrderId:    orderId,
				Code:       txRes.Code,
				Data:       string(bz),
			},
		})
		return true
	})
	return &Block{
		ChainID: header.ChainID,
		CryptoBlock: CryptoBlock{
			BlockHash:   hex.EncodeToString(blockHash),
			ParentHash:  hex.EncodeToString(header.LastBlockId.Hash),
			BlockHeight: header.Height,
			Timestamp:   timeStamp,
			TxTotal:     header.TotalTxs,
			BlockMeta: NativeBlockMeta{
				LastCommitHash:     hex.EncodeToString(header.LastCommitHash),
				DataHash:           hex.EncodeToString(header.DataHash),
				ValidatorsHash:     hex.EncodeToString(header.ValidatorsHash),
				NextValidatorsHash: hex.EncodeToString(header.NextValidatorsHash),
				ConsensusHash:      hex.EncodeToString(header.ConsensusHash),
				AppHash:            hex.EncodeToString(header.AppHash),
				LastResultsHash:    hex.EncodeToString(header.LastResultsHash),
				EvidenceHash:       hex.EncodeToString(header.EvidenceHash),
				ProposerAddress:    sdk.ConsAddress(header.ProposerAddress).String(),
			},
			Transactions: transactionsToPublish,
		},
	}
}

func GetTransferPublished(pool *sdk.Pool, height, blockTime int64) *Transfers {
	transferToPublish := make([]Transfer, 0, 0)
	txs := pool.GetTxs()
	txs.Range(func(key, value interface{}) bool {
		txhash := key.(string)
		stdTx, ok := value.(auth.StdTx)
		var memo string
		if ok {
			memo = stdTx.GetMemo()
		} else {
			Logger.Error("tx is not an auth.StdTx", "hash", txhash)
			return true
		}

		msgs := stdTx.GetMsgs()
		for _, m := range msgs {
			msg, ok := m.(bank.MsgSend)
			if !ok {
				continue
			}
			receivers := make([]Receiver, 0, len(msg.Outputs))
			for _, o := range msg.Outputs {
				coins := make([]Coin, 0, len(o.Coins))
				for _, c := range o.Coins {
					coins = append(coins, Coin{c.Denom, c.Amount})
				}
				receivers = append(receivers, Receiver{Addr: o.Address.String(), Coins: coins})
			}
			transferToPublish = append(transferToPublish, Transfer{TxHash: txhash, Memo: memo, From: msg.Inputs[0].Address.String(), To: receivers})
		}
		return true
	})
	return &Transfers{Height: height, Num: len(transferToPublish), Timestamp: blockTime, Transfers: transferToPublish}
}

func GetAccountBalances(mapper auth.AccountKeeper, ctx sdk.Context, accSlices ...[]string) (res map[string]Account) {
	res = make(map[string]Account)

	for _, accs := range accSlices {
		for _, addrBytesStr := range accs {
			if _, ok := res[addrBytesStr]; !ok {
				addr := sdk.AccAddress([]byte(addrBytesStr))
				if acc, ok := mapper.GetAccount(ctx, addr).(types.NamedAccount); ok {
					assetsMap := make(map[string]*AssetBalance)
					// TODO(#66): set the length to be the total coins this account owned
					assets := make([]*AssetBalance, 0, 10)

					for _, freeCoin := range acc.GetCoins() {
						if assetBalance, ok := assetsMap[freeCoin.Denom]; ok {
							assetBalance.Free = freeCoin.Amount
						} else {
							newAB := &AssetBalance{Asset: freeCoin.Denom, Free: freeCoin.Amount}
							assets = append(assets, newAB)
							assetsMap[freeCoin.Denom] = newAB
						}
					}

					for _, frozenCoin := range acc.GetFrozenCoins() {
						if assetBalance, ok := assetsMap[frozenCoin.Denom]; ok {
							assetBalance.Frozen = frozenCoin.Amount
						} else {
							newAB := &AssetBalance{Asset: frozenCoin.Denom, Frozen: frozenCoin.Amount}
							assets = append(assets, newAB)
							assetsMap[frozenCoin.Denom] = newAB
						}
					}

					for _, lockedCoin := range acc.GetLockedCoins() {
						if assetBalance, ok := assetsMap[lockedCoin.Denom]; ok {
							assetBalance.Locked = lockedCoin.Amount
						} else {
							newAB := &AssetBalance{Asset: lockedCoin.Denom, Locked: lockedCoin.Amount}
							assets = append(assets, newAB)
							assetsMap[lockedCoin.Denom] = newAB
						}
					}

					res[addrBytesStr] = Account{Owner: addrBytesStr, Sequence: acc.GetSequence(), Balances: assets}
				} else {
					Logger.Error(fmt.Sprintf("failed to get account %s from AccountKeeper", addr.String()))
				}
			}
		}
	}

	return
}

func MatchAndAllocateAllForPublish(dexKeeper *orderPkg.DexKeeper, ctx sdk.Context, matchAllMiniSymbols bool) ([]*Trade, []*Trade) {
	// This channels is used for protect not update `dexKeeper.OrderChanges` concurrently
	// matcher would send item to postAlloTransHandler in several goroutine (well-designed)
	// while dexKeeper.OrderChanges are not separated by concurrent factor (users here)
	iocExpireFeeHolderCh := make(chan orderPkg.ExpireHolder, TransferCollectionChannelSize+MiniTransferCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go updateExpireFeeForPublish(dexKeeper, &wg, iocExpireFeeHolderCh)
	var postAlloTransHandler = func(tran orderPkg.Transfer) {
		if tran.IsExpire() {
			if tran.IsExpiredWithFee() {
				// we only got expire of Ioc here, gte orders expire is handled in breathe block
				iocExpireFeeHolderCh <- orderPkg.ExpireHolder{tran.Oid, orderPkg.IocNoFill, tran.Fee.String(), tran.Symbol}
			} else {
				iocExpireFeeHolderCh <- orderPkg.ExpireHolder{tran.Oid, orderPkg.IocExpire, tran.Fee.String(), tran.Symbol}
			}
		}
	}

	dexKeeper.MatchAndAllocateSymbols(ctx, postAlloTransHandler, matchAllMiniSymbols)
	close(iocExpireFeeHolderCh)

	tradeHeight := ctx.BlockHeight()
	tradesToPublish, miniTradesToPublish := extractTradesToPublish(dexKeeper, ctx, tradeHeight)
	wg.Wait()
	return tradesToPublish, miniTradesToPublish
}

func extractTradesToPublish(dexKeeper *orderPkg.DexKeeper, ctx sdk.Context, tradeHeight int64) (tradesToPublish []*Trade, miniTradesToPublish []*Trade) {
	tradesToPublish = make([]*Trade, 0, 32)
	miniTradesToPublish = make([]*Trade, 0, 32)
	tradeIdx := 0

	for symbol := range dexKeeper.GetEngines() {
		matchEngTrades, _ := dexKeeper.GetLastTrades(tradeHeight, symbol)
		for _, trade := range matchEngTrades {
			var ssinglefee string
			var bsinglefee string
			// nilness check is for before Galileo upgrade the trade fee is nil
			if trade.SellerFee != nil {
				ssinglefee = trade.SellerFee.String()
			}
			// nilness check is for before Galileo upgrade the trade fee is nil
			if trade.BuyerFee != nil {
				bsinglefee = trade.BuyerFee.String()
			}

			t := &Trade{
				Id:         fmt.Sprintf("%d-%d", tradeHeight, tradeIdx),
				Symbol:     symbol,
				Sid:        trade.Sid,
				Bid:        trade.Bid,
				Price:      trade.LastPx,
				Qty:        trade.LastQty,
				SSingleFee: ssinglefee,
				BSingleFee: bsinglefee,
				TickType:   int(trade.TickType),
			}
			tradeIdx += 1
			if utils.IsMiniTokenTradingPair(symbol) {
				miniTradesToPublish = append(miniTradesToPublish, t)
			} else {
				tradesToPublish = append(tradesToPublish, t)
			}
		}
	}
	return tradesToPublish, miniTradesToPublish
}

func ExpireOrdersForPublish(
	dexKeeper *orderPkg.DexKeeper,
	ctx sdk.Context,
	blockTime time.Time) {
	expireHolderCh := make(chan orderPkg.ExpireHolder, TransferCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go updateExpireFeeForPublish(dexKeeper, &wg, expireHolderCh)
	var collectorForExpires = func(tran orderPkg.Transfer) {
		if tran.IsExpire() {
			expireHolderCh <- orderPkg.ExpireHolder{tran.Oid, orderPkg.Expired, tran.Fee.String(), tran.Symbol}
		}
	}
	dexKeeper.ExpireOrders(ctx, blockTime, collectorForExpires)
	close(expireHolderCh)
	wg.Wait()
	return
}

func DelistTradingPairForPublish(ctx sdk.Context, dexKeeper *orderPkg.DexKeeper, symbol string) {
	expireHolderCh := make(chan orderPkg.ExpireHolder, TransferCollectionChannelSize)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go updateExpireFeeForPublish(dexKeeper, &wg, expireHolderCh)
	var collectorForExpires = func(tran orderPkg.Transfer) {
		if tran.IsExpire() {
			expireHolderCh <- orderPkg.ExpireHolder{tran.Oid, orderPkg.Expired, tran.Fee.String(), tran.Symbol}
		}
	}
	dexKeeper.DelistTradingPair(ctx, symbol, collectorForExpires)
	close(expireHolderCh)
	wg.Wait()
	return
}

func CollectProposalsForPublish(passed, failed []int64) Proposals {
	totalProposals := len(passed) + len(failed)
	ps := make([]*Proposal, 0, totalProposals)
	for _, p := range passed {
		ps = append(ps, &Proposal{p, Succeed})
	}
	for _, p := range failed {
		ps = append(ps, &Proposal{p, Failed})
	}
	return Proposals{totalProposals, ps}
}

func CollectStakeUpdatesForPublish(unbondingDelegations []stake.UnbondingDelegation) StakeUpdates {
	length := len(unbondingDelegations)
	completedUnbondingDelegations := make([]*CompletedUnbondingDelegation, 0, length)
	for _, ubd := range unbondingDelegations {
		amount := Coin{ubd.Balance.Denom, ubd.Balance.Amount}
		completedUnbondingDelegations = append(completedUnbondingDelegations, &CompletedUnbondingDelegation{ubd.ValidatorAddr, ubd.DelegatorAddr, amount})
	}
	return StakeUpdates{length, completedUnbondingDelegations}
}

func updateExpireFeeForPublish(
	dexKeeper *orderPkg.DexKeeper,
	wg *sync.WaitGroup,
	expHolderCh <-chan orderPkg.ExpireHolder) {
	defer wg.Done()
	for expHolder := range expHolderCh {
		Logger.Debug("transfer collector for order", "orderId", expHolder.OrderId)
		change := orderPkg.OrderChange{expHolder.OrderId, expHolder.Reason, expHolder.Fee, nil}
		dexKeeper.UpdateOrderChangeSync(change, expHolder.Symbol)
	}
}

// collect all changed books according to published order status
func filterChangedOrderBooksByOrders(
	ordersToPublish []*Order,
	latestPriceLevels orderPkg.ChangedPriceLevelsMap, res orderPkg.ChangedPriceLevelsMap) orderPkg.ChangedPriceLevelsMap {
	// map from symbol -> price -> qty diff in this block
	var buyQtyDiff = make(map[string]map[int64]int64)
	var sellQtyDiff = make(map[string]map[int64]int64)
	var allSymbols = make(map[string]struct{})
	for _, o := range ordersToPublish {
		price := o.Price
		symbol := o.Symbol

		if _, ok := latestPriceLevels[symbol]; !ok {
			continue
		}
		allSymbols[symbol] = struct{}{}
		if _, ok := res[symbol]; !ok {
			res[symbol] = orderPkg.ChangedPriceLevelsPerSymbol{make(map[int64]int64), make(map[int64]int64)}
			buyQtyDiff[symbol] = make(map[int64]int64)
			sellQtyDiff[symbol] = make(map[int64]int64)
		}

		switch o.Side {
		case orderPkg.Side.BUY:
			if qty, ok := latestPriceLevels[symbol].Buys[price]; ok {
				res[symbol].Buys[price] = qty
			} else {
				res[symbol].Buys[price] = 0
			}
			buyQtyDiff[symbol][price] += o.effectQtyToOrderBook()
		case orderPkg.Side.SELL:
			if qty, ok := latestPriceLevels[symbol].Sells[price]; ok {
				res[symbol].Sells[price] = qty
			} else {
				res[symbol].Sells[price] = 0
			}
			sellQtyDiff[symbol][price] += o.effectQtyToOrderBook()
		}
	}

	// filter touched but qty actually not changed price levels
	for symbol, priceToQty := range buyQtyDiff {
		for price, qty := range priceToQty {
			if qty == 0 {
				delete(res[symbol].Buys, price)
			}
		}
	}
	for symbol, priceToQty := range sellQtyDiff {
		for price, qty := range priceToQty {
			if qty == 0 {
				delete(res[symbol].Sells, price)
			}
		}
	}
	for symbol := range allSymbols {
		if len(res[symbol].Sells) == 0 && len(res[symbol].Buys) == 0 {
			delete(res, symbol)
		}
	}

	return res
}

func tradeToOrder(t *Trade, o *orderPkg.OrderInfo, timestamp int64, feeHolder orderPkg.FeeHolder, feeToPublish map[string]string) Order {
	var status orderPkg.ChangeType
	if o.CumQty == o.Quantity {
		status = orderPkg.FullyFill
	} else {
		status = orderPkg.PartialFill
	}
	fee := getSerializedFeeForOrder(o, status, feeHolder, feeToPublish)
	owner := o.Sender
	res := Order{
		o.Symbol,
		status,
		o.Id,
		t.Id,
		owner.String(),
		o.Side,
		orderPkg.OrderType.LIMIT,
		o.Price,
		o.Quantity,
		t.Price,
		t.Qty,
		o.CumQty,
		fee,
		o.CreatedTimestamp,
		timestamp,
		o.TimeInForce,
		orderPkg.NEW,
		o.TxHash,
		"",
	}
	if o.Side == orderPkg.Side.BUY {
		res.SingleFee = t.BSingleFee
		t.BAddr = string(owner.Bytes())
		t.Bfee = fee
		t.BSrc = o.TxSource
	} else {
		res.SingleFee = t.SSingleFee
		t.SAddr = string(owner.Bytes())
		t.Sfee = fee
		t.SSrc = o.TxSource
	}
	return res
}

// we collect OrderPart here to make matcheng module independent
func collectOrdersToPublish(
	trades []*Trade,
	orderChanges orderPkg.OrderChanges,
	orderInfos orderPkg.OrderInfoForPublish,
	feeHolder orderPkg.FeeHolder,
	timestamp int64, miniTrades []*Trade,
	miniOrderChanges orderPkg.OrderChanges,
	miniOrderInfos orderPkg.OrderInfoForPublish) (opensToPublish []*Order, closedToPublish []*Order, miniOpensToPublish []*Order, miniClosedToPublish []*Order, feeToPublish map[string]string) {
	opensToPublish = make([]*Order, 0)
	closedToPublish = make([]*Order, 0)
	miniOpensToPublish = make([]*Order, 0)
	miniClosedToPublish = make([]*Order, 0)
	// serve as a cache to avoid fee's serialization several times for one address
	feeToPublish = make(map[string]string)

	// the following two maps are used to update fee field we published
	// more detail can be found at:
	// https://github.com/binance-chain/docs-site/wiki/Fee-Calculation,-Collection-and-Distribution#publication
	chargedCancels := make(map[string]int)
	chargedExpires := make(map[string]int)

	// collect orders (new, cancel, ioc-no-fill, expire, failed-blocking and failed-matching) from orderChanges
	opensToPublish, closedToPublish = collectOrders(orderChanges, orderInfos, timestamp, opensToPublish, closedToPublish, chargedCancels, chargedExpires)
	miniOpensToPublish, miniClosedToPublish = collectOrders(miniOrderChanges, miniOrderInfos, timestamp, miniOpensToPublish, miniClosedToPublish, chargedCancels, chargedExpires)

	// update C and E fields in serialized fee string
	updateCancelExpireOrderNum(closedToPublish, orderInfos, feeToPublish, chargedCancels, chargedExpires, feeHolder)
	updateCancelExpireOrderNum(miniClosedToPublish, miniOrderInfos, feeToPublish, chargedCancels, chargedExpires, feeHolder)
	// update fee and collect orders from trades
	opensToPublish, closedToPublish = convertTradesToOrders(trades, orderInfos, timestamp, feeHolder, feeToPublish, opensToPublish, closedToPublish)
	miniOpensToPublish, miniClosedToPublish = convertTradesToOrders(miniTrades, miniOrderInfos, timestamp, feeHolder, feeToPublish, miniOpensToPublish, miniClosedToPublish)

	return opensToPublish, closedToPublish, miniOpensToPublish, miniClosedToPublish, feeToPublish
}

func convertTradesToOrders(trades []*Trade, orderInfos orderPkg.OrderInfoForPublish, timestamp int64, feeHolder orderPkg.FeeHolder, feeToPublish map[string]string, opensToPublish []*Order, closedToPublish []*Order) ([]*Order, []*Order) {
	for _, t := range trades {
		if o, exists := orderInfos[t.Bid]; exists {
			orderToPublish := tradeToOrder(t, o, timestamp, feeHolder, feeToPublish)
			if orderToPublish.Status.IsOpen() {
				opensToPublish = append(opensToPublish, &orderToPublish)
			} else {
				closedToPublish = append(closedToPublish, &orderToPublish)
			}
		} else {
			Logger.Error("failed to resolve order information from orderInfos", "orderId", t.Bid)
		}

		if o, exists := orderInfos[t.Sid]; exists {
			orderToPublish := tradeToOrder(t, o, timestamp, feeHolder, feeToPublish)
			if orderToPublish.Status.IsOpen() {
				opensToPublish = append(opensToPublish, &orderToPublish)
			} else {
				closedToPublish = append(closedToPublish, &orderToPublish)
			}
		} else {
			Logger.Error("failed to resolve order information from orderInfos", "orderId", t.Sid)
		}
	}
	return opensToPublish, closedToPublish
}

func updateCancelExpireOrderNum(closedToPublish []*Order, orderInfos orderPkg.OrderInfoForPublish, feeToPublish map[string]string, chargedCancels map[string]int, chargedExpires map[string]int, feeHolder orderPkg.FeeHolder) {
	for _, order := range closedToPublish {
		if orderInfo, ok := orderInfos[order.OrderId]; ok {
			senderBytesStr := string(orderInfo.Sender)
			if _, ok := feeToPublish[senderBytesStr]; !ok {
				numOfChargedCanceled := chargedCancels[senderBytesStr]
				numOfExpiredCanceled := chargedExpires[senderBytesStr]
				if raw, ok := feeHolder[senderBytesStr]; ok {
					fee := raw.SerializeForPub(numOfChargedCanceled, numOfExpiredCanceled)
					feeToPublish[senderBytesStr] = fee
					order.Fee = fee
				} else if numOfChargedCanceled > 0 || numOfExpiredCanceled > 0 {
					Logger.Error("cannot find fee for cancel/expire", "sender", order.Owner)
				}
			}
		} else {
			Logger.Error("should not to locate order in OrderChangesMap", "oid", order.OrderId)
		}
	}
}

func collectOrders(orderChanges orderPkg.OrderChanges, orderInfos orderPkg.OrderInfoForPublish, timestamp int64, opensToPublish []*Order, closedToPublish []*Order, chargedCancels map[string]int, chargedExpires map[string]int) ([]*Order, []*Order) {
	for _, o := range orderChanges {
		if orderInfo := o.ResolveOrderInfo(orderInfos); orderInfo != nil {
			orderToPublish := Order{
				orderInfo.Symbol, o.Tpe, o.Id,
				"", orderInfo.Sender.String(), orderInfo.Side,
				orderPkg.OrderType.LIMIT, orderInfo.Price, orderInfo.Quantity,
				0, 0, orderInfo.CumQty, "",
				orderInfo.CreatedTimestamp, timestamp, orderInfo.TimeInForce,
				orderPkg.NEW, orderInfo.TxHash, o.SingleFee,
			}

			if o.Tpe.IsOpen() {
				opensToPublish = append(opensToPublish, &orderToPublish)
			} else {
				closedToPublish = append(closedToPublish, &orderToPublish)
			}

			// fee field handling
			if orderToPublish.isChargedCancel() {
				if _, ok := chargedCancels[string(orderInfo.Sender)]; ok {
					chargedCancels[string(orderInfo.Sender)] += 1
				} else {
					chargedCancels[string(orderInfo.Sender)] = 1
				}
			} else if orderToPublish.isChargedExpire() {
				if _, ok := chargedExpires[string(orderInfo.Sender)]; ok {
					chargedExpires[string(orderInfo.Sender)] += 1
				} else {
					chargedExpires[string(orderInfo.Sender)] = 1
				}
			}
		} else {
			Logger.Error("failed to locate order change in OrderChangesMap", "orderChange", o.String())
		}
	}
	return opensToPublish, closedToPublish
}

func getSerializedFeeForOrder(orderInfo *orderPkg.OrderInfo, status orderPkg.ChangeType, feeHolder orderPkg.FeeHolder, feeToPublish map[string]string) string {
	senderStr := string(orderInfo.Sender)

	// if the serialized fee has been cached, return it directly
	if cached, ok := feeToPublish[senderStr]; ok {
		return cached
	} else {
		feeStr := ""
		if fee, ok := feeHolder[senderStr]; ok {
			feeStr = fee.String()
			feeToPublish[senderStr] = feeStr
		} else {
			if orderInfo.CumQty == 0 && status != orderPkg.Ack {
				Logger.Error("cannot find fee from fee holder", "orderId", orderInfo.Id)
			}
		}
		return feeStr
	}

}

func transferInputsToPublish(inputs []bank.Input) []Input {
	transInputs := make([]Input, 0, len(inputs))
	for _, i := range inputs {
		input := Input{Address: i.Address.String()}
		for _, c := range i.Coins {
			input.Coins = append(input.Coins, Coin{Denom: c.Denom, Amount: c.Amount})
		}
		transInputs = append(transInputs, input)
	}
	return transInputs
}

func transferOutputsToPublish(outputs []bank.Output) []Output {
	transOutputs := make([]Output, 0, len(outputs))
	for _, o := range outputs {
		output := Output{Address: o.Address.String()}
		for _, c := range o.Coins {
			output.Coins = append(output.Coins, Coin{Denom: c.Denom, Amount: c.Amount})
		}
		transOutputs = append(transOutputs, output)
	}
	return transOutputs
}
