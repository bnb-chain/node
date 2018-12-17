package order

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/fees"
	"github.com/BiJie/BinanceChain/common/log"
	common "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

type NewOrderResponse struct {
	OrderID string `json:"order_id"`
}

// NewHandler - returns a handler for dex type messages.
func NewHandler(cdc *wire.Codec, k *Keeper, accKeeper auth.AccountKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case NewOrderMsg:
			return handleNewOrder(ctx, cdc, k, msg, simulate)
		case CancelOrderMsg:
			return handleCancelOrder(ctx, k, msg, simulate)
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
		return fmt.Errorf("the order ID given did not match the expected one: `%s`", expectedID)
	}

	pair, err := pairMapper.GetTradingPair(ctx, baseAsset, quoteAsset)
	if err != nil {
		return err
	}

	if msg.Quantity <= 0 || msg.Quantity%pair.LotSize.ToInt64() != 0 {
		return errors.New(fmt.Sprintf("quantity(%v) is not rounded to lotSize(%v)", msg.Quantity, pair.LotSize))
	}

	if msg.Price <= 0 || msg.Price%pair.TickSize.ToInt64() != 0 {
		return errors.New(fmt.Sprintf("price(%v) is not rounded to tickSize(%v)", msg.Price, pair.TickSize))
	}

	if utils.IsExceedMaxNotional(msg.Price, msg.Quantity) {
		return errors.New("notional value of the order is too large(cannot fit in int64)")
	}

	return nil
}

func handleNewOrder(
	ctx sdk.Context, cdc *wire.Codec, keeper *Keeper, msg NewOrderMsg, simulate bool,
) sdk.Result {
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

	// TODO: the below is mostly copied from FreezeToken. It should be rewritten once "locked" becomes a field on account
	// this is done in memory! we will run this block in checktx/simulate
	if ctx.IsCheckTx() || simulate {
		logger.Info("Incoming New Order", "order", msg)
		//only check whether there exists order to cancel
		if _, ok := keeper.OrderExists(msg.Symbol, msg.Id); ok {
			errString := fmt.Sprintf("Duplicated order [%v] on symbol [%v]", msg.Id, msg.Symbol)
			return sdk.NewError(types.DefaultCodespace, types.CodeDuplicatedOrder, errString).Result()
		}
	}

	// the following is done in the app's checkstate / deliverstate, so it's safe to ignore isCheckTx
	var amountToLock int64
	baseAsset, quoteAsset := utils.TradingPair2AssetsSafe(msg.Symbol)
	var symbolToLock string
	if msg.Side == Side.BUY {
		// TODO: where is 10^8 stored?
		amountToLock = utils.CalBigNotional(msg.Quantity, msg.Price)
		symbolToLock = strings.ToUpper(quoteAsset)
	} else {
		amountToLock = msg.Quantity
		symbolToLock = strings.ToUpper(baseAsset)
	}
	freeBalance := acc.GetCoins()
	if freeBalance.AmountOf(symbolToLock) < amountToLock {
		return sdk.ErrInsufficientCoins("do not have enough token to lock").Result()
	}

	toLockCoins := sdk.Coins{{Denom: symbolToLock, Amount: amountToLock}}
	// the following is done in the app's checkstate / deliverstate, so it's safe to ignore isCheckTx
	// TODO: perform reduce avail + increase locked + insert orderbook atomically
	acc.SetCoins(freeBalance.Minus(toLockCoins))
	acc.SetLockedCoins(acc.GetLockedCoins().Plus(toLockCoins))
	keeper.am.SetAccount(ctx, acc)

	// this is done in memory! we must not run this block in checktx or simulate!
	if !ctx.IsCheckTx() && !simulate { // only subtract coins & insert into OB during DeliverTx
		if txHash, ok := ctx.Value(common.TxHashKey).(string); ok {
			height := ctx.BlockHeader().Height
			timestamp := ctx.BlockHeader().Time.Unix()
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
			return sdk.NewError(
				types.DefaultCodespace,
				types.CodeFailInsertOrder,
				"cannot get txHash from ctx").Result()
		}
	}

	response := NewOrderResponse{
		OrderID: msg.Id,
	}
	serialized, err := cdc.MarshalJSON(&response)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	return sdk.Result{
		Data: serialized,
	}
}

// Handle CancelOffer -
func handleCancelOrder(
	ctx sdk.Context, keeper *Keeper, msg CancelOrderMsg, simulate bool,
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
	acc := keeper.am.GetAccount(ctx, msg.Sender)
	fee := keeper.FeeManager.CalcFixedFee(acc.GetCoins(), transfer.eventType, transfer.inAsset, keeper.lastTradePrices)
	acc.SetCoins(acc.GetCoins().Minus(fee.Tokens))
	keeper.am.SetAccount(ctx, acc)

	// this is done in memory! we must not run this block in checktx or simulate!
	if ctx.IsDeliverTx() {
		//remove order from cache and order book
		err := keeper.RemoveOrder(origOrd.Id, origOrd.Symbol, func(ord me.OrderPart) {
			if keeper.CollectOrderInfoForPublish {
				change := OrderChange{msg.Id, Canceled}
				keeper.OrderChanges = append(keeper.OrderChanges, change)
				keeper.updateRoundOrderFee(string(msg.Sender), fee)
			}
		})
		if err != nil {
			return sdk.NewError(types.DefaultCodespace, types.CodeFailCancelOrder, err.Error()).Result()
		}

		fees.Pool.AddFee(fee)
	}

	return sdk.Result{}
}
