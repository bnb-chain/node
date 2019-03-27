package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/log"
	common "github.com/binance-chain/node/common/types"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/wire"
)

type NewOrderResponse struct {
	OrderID string `json:"order_id"`
}

// NewHandler - returns a handler for dex type messages.
func NewHandler(cdc *wire.Codec, k *Keeper, accKeeper auth.AccountKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case NewOrderMsg:
			return handleNewOrder(ctx, cdc, k, msg)
		case CancelOrderMsg:
			return handleCancelOrder(ctx, k, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func validateOrder(ctx sdk.Context, pairMapper store.TradingPairMapper, acc sdk.Account, msg NewOrderMsg) error {
	baseAsset, quoteAsset, err := utils.TradingPair2Assets(msg.Symbol)
	if err != nil {
		return err
	}

	seq := acc.GetSequence()
	expectedID := GenerateOrderID(seq, msg.Sender)
	if expectedID != msg.Id {
		return fmt.Errorf("the order ID(%s) given did not match the expected one: `%s`", msg.Id, expectedID)
	}

	pair, err := pairMapper.GetTradingPair(ctx, baseAsset, quoteAsset)
	if err != nil {
		return err
	}

	if msg.Quantity <= 0 || msg.Quantity%pair.LotSize.ToInt64() != 0 {
		return errors.New(fmt.Sprintf("quantity(%v) is not rounded to lotSize(%v)", msg.Quantity, pair.LotSize.ToInt64()))
	}

	if msg.Price <= 0 || msg.Price%pair.TickSize.ToInt64() != 0 {
		return errors.New(fmt.Sprintf("price(%v) is not rounded to tickSize(%v)", msg.Price, pair.TickSize.ToInt64()))
	}

	if utils.IsExceedMaxNotional(msg.Price, msg.Quantity) {
		return errors.New("notional value of the order is too large(cannot fit in int64)")
	}

	return nil
}

func validateQtyAndLockBalance(ctx sdk.Context, keeper *Keeper, acc common.NamedAccount, msg NewOrderMsg) error {
	symbol := strings.ToUpper(msg.Symbol)
	baseAssetSymbol, quoteAssetSymbol := utils.TradingPair2AssetsSafe(symbol)
	notional := utils.CalBigNotionalInt64(msg.Price, msg.Quantity)

	// note: the check sequence is well designed.
	freeBalance := acc.GetCoins()
	var toLockCoins sdk.Coins
	if msg.Side == Side.BUY {
		// for buy orders,
		// 1. total notional == ToLock(quoteAsset) <= FreeBalance(quoteAsset) <= TotalSupply(quoteAsset) < Max(int64)
		// 2. check whether the qty on this price level will overflow.

		if freeBalance.AmountOf(quoteAssetSymbol) < notional {
			return errors.New("do not have enough token to lock")
		}

		pl := keeper.GetPriceLevel(symbol, msg.Side, msg.Price)
		totalQty := msg.Quantity
		if pl != nil {
			totalQty += pl.TotalLeavesQty()
		}
		if totalQty < 0 {
			// overflow, this is a implicit requirement from the match engine.
			return errors.New("order quantity is too large to be placed on this price level")
		}

		toLockCoins = sdk.Coins{{Denom: quoteAssetSymbol, Amount: notional}}
	} else {
		// for sell orders,
		// 1. total qty == ToLock(baseAsset) <= FreeBalance(baseAsset) <= TotalSupply(baseAsset) < Max(int64)
		// 2. For a sell order, total notional on one price level is allowed to overflow.
		// This order won't be fully filled as the buyer does not have such huge tokens to pay for it.

		if freeBalance.AmountOf(baseAssetSymbol) < msg.Quantity {
			return errors.New("do not have enough token to lock")
		}

		toLockCoins = sdk.Coins{{Denom: baseAssetSymbol, Amount: msg.Quantity}}
	}

	acc.SetCoins(freeBalance.Minus(toLockCoins))
	acc.SetLockedCoins(acc.GetLockedCoins().Plus(toLockCoins))
	keeper.am.SetAccount(ctx, acc)
	return nil
}

func handleNewOrder(
	ctx sdk.Context, cdc *wire.Codec, keeper *Keeper, msg NewOrderMsg,
) sdk.Result {
	// TODO: the below is mostly copied from FreezeToken. It should be rewritten once "locked" becomes a field on account
	log.With("module", "dex").Info("Incoming New Order", "order", msg)
	// this check costs least.
	if _, ok := keeper.OrderExists(msg.Symbol, msg.Id); ok {
		errString := fmt.Sprintf("Duplicated order [%v] on symbol [%v]", msg.Id, msg.Symbol)
		return sdk.NewError(types.DefaultCodespace, types.CodeDuplicatedOrder, errString).Result()
	}

	acc := keeper.am.GetAccount(ctx, msg.Sender).(common.NamedAccount)
	if !ctx.IsReCheckTx() {
		//for recheck:
		// 1. sequence is verified in anteHandler
		// 2. since sequence is verified correct again, id should be ok too
		// 3. trading pair is verified
		// 4. price/qty may have odd tick size/lot size, but it can be handled as
		//    other existing orders.
		err := validateOrder(ctx, keeper.PairMapper, acc, msg)
		if err != nil {
			return sdk.NewError(types.DefaultCodespace, types.CodeInvalidOrderParam, err.Error()).Result()
		}
	}

	// the following is done in the app's checkstate / deliverstate, so it's safe to ignore isCheckTx
	err := validateQtyAndLockBalance(ctx, keeper, acc, msg)
	if err != nil {
		return sdk.NewError(types.DefaultCodespace, types.CodeInvalidOrderParam, err.Error()).Result()
	}

	// this is done in memory! we must not run this block in checktx or simulate!
	if ctx.IsDeliverTx() { // only subtract coins & insert into OB during DeliverTx
		if txHash, ok := ctx.Value(baseapp.TxHashKey).(string); ok {
			blockHeader := ctx.BlockHeader()
			height := blockHeader.Height
			timestamp := blockHeader.Time.Unix()
			msg := OrderInfo{
				msg,
				height, timestamp,
				height, timestamp,
				0, txHash}
			err := keeper.AddOrder(msg, false)
			if err != nil {
				return sdk.NewError(types.DefaultCodespace, types.CodeFailInsertOrder, err.Error()).Result()
			}
		} else {
			panic("cannot get txHash from ctx")
		}
	}

	response := NewOrderResponse{
		OrderID: msg.Id,
	}
	serialized, err := json.Marshal(&response)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	return sdk.Result{
		Data: serialized,
	}
}

// Handle CancelOffer -
func handleCancelOrder(
	ctx sdk.Context, keeper *Keeper, msg CancelOrderMsg,
) sdk.Result {
	origOrd, ok := keeper.OrderExists(msg.Symbol, msg.RefId)

	//only check whether there exists order to cancel
	if !ok {
		errString := fmt.Sprintf("Failed to find order [%v]", msg.RefId)
		return sdk.NewError(types.DefaultCodespace, types.CodeFailLocateOrderToCancel, errString).Result()
	}

	// only can cancel their own order
	if !reflect.DeepEqual(msg.Sender, origOrd.Sender) {
		errString := fmt.Sprintf("Order [%v] does not belong to transaction sender", msg.RefId)
		return sdk.NewError(types.DefaultCodespace, types.CodeFailLocateOrderToCancel, errString).Result()
	}

	log.With("module", "dex").Info("Incoming Cancel", "cancel", msg)
	ord, err := keeper.GetOrder(origOrd.Id, origOrd.Symbol, origOrd.Side, origOrd.Price)
	if err != nil {
		return sdk.NewError(types.DefaultCodespace, types.CodeFailLocateOrderToCancel, err.Error()).Result()
	}
	transfer := TransferFromCanceled(ord, origOrd, false)
	sdkError := keeper.doTransfer(ctx, &transfer)
	if sdkError != nil {
		return sdkError.Result()
	}
	fee := common.Fee{}
	if !transfer.FeeFree() {
		acc := keeper.am.GetAccount(ctx, msg.Sender)
		fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), transfer.eventType, transfer.inAsset, keeper.engines)
		acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
		keeper.am.SetAccount(ctx, acc)
	}

	// this is done in memory! we must not run this block in checktx or simulate!
	if ctx.IsDeliverTx() {
		if txHash, ok := ctx.Value(baseapp.TxHashKey).(string); !ok {
			panic("cannot get txHash from ctx")
		} else {
			// add fee to pool, even it's free
			fees.Pool.AddFee(txHash, fee)
		}
		//remove order from cache and order book
		err := keeper.RemoveOrder(origOrd.Id, origOrd.Symbol, func(ord me.OrderPart) {
			if keeper.CollectOrderInfoForPublish {
				change := OrderChange{msg.RefId, Canceled, nil}
				keeper.OrderChanges = append(keeper.OrderChanges, change)
				keeper.updateRoundOrderFee(string(msg.Sender), fee)
			}
		})
		if err != nil {
			return sdk.NewError(types.DefaultCodespace, types.CodeFailCancelOrder, err.Error()).Result()
		}
	}

	return sdk.Result{}
}
