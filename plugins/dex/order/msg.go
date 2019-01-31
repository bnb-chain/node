package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"

	"github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/types"
)

const (
	RouteNewOrder    = "orderNew"
	RouteCancelOrder = "orderCancel"
)

// Side/TimeInForce/OrderType are const, following FIX protocol convention
// Used as Enum
var Side = struct {
	BUY  int8
	SELL int8
}{matcheng.BUYSIDE, matcheng.SELLSIDE}

var sideNames = map[string]int8{
	"BUY":  matcheng.BUYSIDE,
	"SELL": matcheng.SELLSIDE,
}

// GenerateOrderID generates an order ID
func GenerateOrderID(sequence int64, addr sdk.AccAddress) string {
	id := fmt.Sprintf("%X-%d", addr, sequence)
	return id
}

// IsValidSide validates that a side is valid and supported by the matching engine
func IsValidSide(side int8) bool {
	switch side {
	case Side.BUY, Side.SELL:
		return true
	default:
		return false
	}
}

// SideStringToSideCode converts a string like "BUY" to its internal side code
func SideStringToSideCode(side string) (int8, error) {
	upperSide := strings.ToUpper(side)
	if val, ok := sideNames[upperSide]; ok {
		return val, nil
	}
	return -1, errors.New("side `" + upperSide + "` not found or supported")
}

const (
	_           int8 = iota
	orderMarket int8 = iota
	orderLimit  int8 = iota
)

// OrderType is an enum of order type options supported by the matching engine
var OrderType = struct {
	LIMIT  int8
	MARKET int8
}{orderLimit, orderMarket}

// IsValidOrderType validates that an order type is valid and supported by the matching engine
func IsValidOrderType(ot int8) bool {
	switch ot {
	case OrderType.LIMIT: // only allow LIMIT for now.
		return true
	default:
		return false
	}
}

const (
	_      int8 = iota
	tifGTE int8 = iota
	_      int8 = iota
	tifIOC int8 = iota
)

// TimeInForce is an enum of TIF (Time in Force) options supported by the matching engine
var TimeInForce = struct {
	GTE int8
	IOC int8
}{tifGTE, tifIOC}

var timeInForceNames = map[string]int8{
	"GTE": tifGTE,
	"IOC": tifIOC,
}

// IsValidTimeInForce validates that a tif code is correct
func IsValidTimeInForce(tif int8) bool {
	switch tif {
	case TimeInForce.GTE, TimeInForce.IOC:
		return true
	default:
		return false
	}
}

// TifStringToTifCode converts a string like "GTE" to its internal tif code
func TifStringToTifCode(tif string) (int8, error) {
	upperTif := strings.ToUpper(tif)
	if val, ok := timeInForceNames[upperTif]; ok {
		return val, nil
	}
	return -1, errors.New("tif `" + upperTif + "` not found or supported")
}

var _ sdk.Msg = NewOrderMsg{}

type NewOrderMsg struct {
	Sender      sdk.AccAddress `json:"sender"`
	Id          string         `json:"id"`
	Symbol      string         `json:"symbol"`
	OrderType   int8           `json:"ordertype"`
	Side        int8           `json:"side"`
	Price       int64          `json:"price"`
	Quantity    int64          `json:"quantity"`
	TimeInForce int8           `json:"timeinforce"`
}

// NewNewOrderMsg constructs a new NewOrderMsg
func NewNewOrderMsg(sender sdk.AccAddress, id string, side int8,
	symbol string, price int64, qty int64) NewOrderMsg {
	return NewOrderMsg{
		Sender:      sender,
		Id:          id,
		Symbol:      symbol,
		OrderType:   OrderType.LIMIT, // default
		Side:        side,
		Price:       price,
		Quantity:    qty,
		TimeInForce: TimeInForce.GTE, // default
	}
}

// NewNewOrderMsgAuto constructs a new NewOrderMsg and auto-assigns its order ID
func NewNewOrderMsgAuto(txBuilder txbuilder.TxBuilder, sender sdk.AccAddress, side int8,
	symbol string, price int64, qty int64) (NewOrderMsg, error) {
	var id string
	ctx, err := context.EnsureSequence(ctx)
	if err != nil {
		return NewOrderMsg{}, err
	}
	id = GenerateOrderID(ctx.Sequence, sender)
	return NewOrderMsg{
		Sender:      sender,
		Id:          id,
		Symbol:      symbol,
		OrderType:   OrderType.LIMIT, // default
		Side:        side,
		Price:       price,
		Quantity:    qty,
		TimeInForce: TimeInForce.GTE, // default
	}, nil
}

// nolint
func (msg NewOrderMsg) Route() string                { return RouteNewOrder }
func (msg NewOrderMsg) Type() string                 { return RouteNewOrder }
func (msg NewOrderMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.Sender} }
func (msg NewOrderMsg) String() string {
	return fmt.Sprintf("NewOrderMsg{Sender: %v, Id: %v, Symbol: %v}", msg.Sender, msg.Id, msg.Symbol)
}
func (msg NewOrderMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

type OrderInfo struct {
	NewOrderMsg
	CreatedHeight        int64
	CreatedTimestamp     int64
	LastUpdatedHeight    int64
	LastUpdatedTimestamp int64
	CumQty               int64
	TxHash               string
}

var _ sdk.Msg = CancelOrderMsg{}

// CancelOrderMsg represents a message to cancel an open order
type CancelOrderMsg struct {
	Sender sdk.AccAddress `json:"sender"`
	Symbol string         `json:"symbol"`
	RefId  string         `json:"refid"`
}

// NewCancelOrderMsg constructs a new CancelOrderMsg
func NewCancelOrderMsg(sender sdk.AccAddress, symbol, refId string) CancelOrderMsg {
	return CancelOrderMsg{
		Sender: sender,
		Symbol: symbol,
		RefId:  refId,
	}
}

// nolint
func (msg CancelOrderMsg) Route() string                { return RouteCancelOrder }
func (msg CancelOrderMsg) Type() string                 { return RouteCancelOrder }
func (msg CancelOrderMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.Sender} }
func (msg CancelOrderMsg) String() string {
	return fmt.Sprintf("CancelOrderMsg{Sender:%v, RefId: %s}", msg.Sender, msg.RefId)
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg NewOrderMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

// GetSignBytes - Get the bytes for the message signer to sign on
func (msg CancelOrderMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg CancelOrderMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg NewOrderMsg) ValidateBasic() sdk.Error {
	// `-` is required in the compound order id: <address>-<sequence>
	// NOTE: the actual validation of the ID happens in the AnteHandler for now.
	if len(msg.Id) == 0 || !strings.Contains(msg.Id, "-") {
		return types.ErrInvalidOrderParam("Id", fmt.Sprintf("Invalid order ID:%s", msg.Id))
	}
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	if msg.Quantity <= 0 {
		return types.ErrInvalidOrderParam("Quantity", fmt.Sprintf("Zero/Negative Number:%d", msg.Quantity))
	}
	if msg.Price <= 0 {
		return types.ErrInvalidOrderParam("Price", fmt.Sprintf("Zero/Negative Number:%d", msg.Quantity))
	}
	if !IsValidOrderType(msg.OrderType) {
		return types.ErrInvalidOrderParam("OrderType", fmt.Sprintf("Invalid order type:%d", msg.OrderType))
	}
	if !IsValidSide(msg.Side) {
		return types.ErrInvalidOrderParam("Side", fmt.Sprintf("Invalid side:%d", msg.Side))
	}
	if !IsValidTimeInForce(msg.TimeInForce) {
		return types.ErrInvalidOrderParam("TimeInForce", fmt.Sprintf("Invalid TimeInForce:%d", msg.TimeInForce))
	}

	return nil
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg CancelOrderMsg) ValidateBasic() sdk.Error {
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	if len(msg.RefId) == 0 || !strings.Contains(msg.RefId, "-") {
		return types.ErrInvalidOrderParam("RefId", fmt.Sprintf("Invalid ref ID:%s", msg.RefId))
	}
	return nil
}
