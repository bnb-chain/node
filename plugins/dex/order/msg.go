package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"

	"github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
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

func IToSide(side int8) string {
	switch side {
	case Side.BUY:
		return "BUY"
	case Side.SELL:
		return "SELL"
	default:
		return "UNKNOWN"
	}
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

func IToOrderType(tpe int8) string {
	switch tpe {
	case OrderType.LIMIT:
		return "LIMIT"
	case OrderType.MARKET:
		return "MARKET"
	default:
		return "UNKNOWN"
	}
}

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
	tifGTC int8 = iota
	_      int8 = iota
	tifIOC int8 = iota
)

// TimeInForce is an enum of TIF (Time in Force) options supported by the matching engine
var TimeInForce = struct {
	GTC int8
	IOC int8
}{tifGTC, tifIOC}

var timeInForceNames = map[string]int8{
	"GTC": tifGTC,
	"IOC": tifIOC,
}

// IsValidTimeInForce validates that a tif code is correct
func IsValidTimeInForce(tif int8) bool {
	switch tif {
	case TimeInForce.GTC, TimeInForce.IOC:
		return true
	default:
		return false
	}
}

func IToTimeInForce(tif int8) string {
	switch tif {
	case TimeInForce.GTC:
		return "GTC"
	case TimeInForce.IOC:
		return "IOC"
	default:
		return "UNKNOWN"
	}
}

// TifStringToTifCode converts a string like "GTC" to its internal tif code
func TifStringToTifCode(tif string) (int8, error) {
	upperTif := strings.ToUpper(tif)
	if val, ok := timeInForceNames[upperTif]; ok {
		return val, nil
	}
	return -1, errors.New("tif `" + upperTif + "` not found or supported")
}

var _ sdk.Msg = NewOrderMsg{}

type NewOrderMsg struct {
	Version     byte           `json:"version"`
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
		Version:     0x01,
		Sender:      sender,
		Id:          id,
		Symbol:      symbol,
		OrderType:   OrderType.LIMIT, // default
		Side:        side,
		Price:       price,
		Quantity:    qty,
		TimeInForce: TimeInForce.GTC, // default
	}
}

// NewNewOrderMsgAuto constructs a new NewOrderMsg and auto-assigns its order ID
func NewNewOrderMsgAuto(txBuilder txbuilder.TxBuilder, sender sdk.AccAddress, side int8,
	symbol string, price int64, qty int64) (NewOrderMsg, error) {
	var id string
	id = GenerateOrderID(txBuilder.Sequence+1, sender)
	return NewOrderMsg{
		Version:     0x01,
		Sender:      sender,
		Id:          id,
		Symbol:      symbol,
		OrderType:   OrderType.LIMIT, // default
		Side:        side,
		Price:       price,
		Quantity:    qty,
		TimeInForce: TimeInForce.GTC, // default
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
	Version byte `json:"version"`
	Sender  sdk.AccAddress
	Symbol  string `json:"symbol"`
	Id      string `json:"id"`
	RefId   string `json:"refid"`
}

// NewCancelOrderMsg constructs a new CancelOrderMsg
func NewCancelOrderMsg(sender sdk.AccAddress, symbol, id, refId string) CancelOrderMsg {
	return CancelOrderMsg{
		Version: 0x01,
		Sender:  sender,
		Symbol:  symbol,
		Id:      id,
		RefId:   refId,
	}
}

// nolint
func (msg CancelOrderMsg) Route() string                { return RouteCancelOrder }
func (msg CancelOrderMsg) Type() string                 { return RouteCancelOrder }
func (msg CancelOrderMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.Sender} }
func (msg CancelOrderMsg) String() string {
	return fmt.Sprintf("CancelOrderMsg{Sender: %v}", msg.Sender)
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
	if msg.Version != 0x01 {
		// TODO: use a dedicated error type
		return sdk.ErrInternal("Invalid version. Expected 0x01")
	}
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	// `-` is required in the compound order id: <address>-<sequence>
	if len(msg.Id) == 0 || !strings.Contains(msg.Id, "-") {
		return types.ErrInvalidOrderParam("Id", fmt.Sprintf("Invalid order ID:%s", msg.Id))
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
	if msg.Version != 0x01 {
		// TODO: use a dedicated error type
		return sdk.ErrInternal("Invalid version. Expected 0x01")
	}
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	if len(msg.Id) == 0 || !strings.Contains(msg.Id, "-") {
		return types.ErrInvalidOrderParam("Id", fmt.Sprintf("Invalid order ID:%s", msg.Id))
	}
	return nil
}
