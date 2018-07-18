package order

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
)

const Route = "dexOrder"

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

// Side/TimeInForce/OrderType are const, following FIX protocol convention
// Used as Enum
var Side = struct {
	BUY  int8
	SELL int8
}{1, 2}

func IsValidSide(side int8) bool {
	switch side {
	case Side.BUY, Side.SELL:
		return true
	default:
		return false
	}
}

var OrderType = struct {
	LIMIT  int8
	MARKET int8
}{2, 1}

func IsValidOrderType(ot int8) bool {
	switch ot {
	case OrderType.LIMIT: // only allow LIMIT for now.
		return true
	default:
		return false
	}
}

var TimeInForce = struct {
	GTC int8
	IOC int8
}{1, 3}

func IsValidTimeInForce(tif int8) bool {
	switch tif {
	case TimeInForce.GTC, TimeInForce.IOC:
		return true
	default:
		return false
	}
}

type CancelOrderMsg struct {
	Sender sdk.AccAddress
	Id     string `json:"id"`
	Symbol string `json:"symbol"`
	Side   int8   `json:"side"`
	Price  int64  `json:"price"`
}

// NewNewOrderMsg - Creates a new NewOrderMsg
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
		TimeInForce: TimeInForce.GTC, // default
	}
}

var _ sdk.Msg = NewOrderMsg{}

// nolint
func (msg NewOrderMsg) Type() string                            { return Route }
func (msg NewOrderMsg) Get(key interface{}) (value interface{}) { return nil }
func (msg NewOrderMsg) GetSigners() []sdk.AccAddress            { return []sdk.AccAddress{msg.Sender} }
func (msg NewOrderMsg) String() string {
	return fmt.Sprintf("NewOrderMsg{Sender: %v, Id: %v, Symbol: %v}", msg.Sender, msg.Id, msg.Symbol)
}

// NewCancelOrderMsg - Creates a new CancelOrderMsg
func NewCancelOrderMsg(sender sdk.AccAddress, id string) CancelOrderMsg {
	return CancelOrderMsg{
		Sender: sender,
		Id:     id,
	}
}

var _ sdk.Msg = CancelOrderMsg{}

// nolint
func (msg CancelOrderMsg) Type() string                            { return Route }
func (msg CancelOrderMsg) Get(key interface{}) (value interface{}) { return nil }
func (msg CancelOrderMsg) GetSigners() []sdk.AccAddress            { return []sdk.AccAddress{msg.Sender} }
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

//TODO: check whether the symbol is listed or not
func ValidateSymbol(symbol string) error {
	_, _, err := utils.TradeSymbol2Ccy(symbol)
	if err != nil {
		return err
	}
	return nil
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg NewOrderMsg) ValidateBasic() sdk.Error {
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	err := ValidateSymbol(msg.Symbol)
	if err != nil {
		return types.ErrInvalidTradeSymbol(err.Error())
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
	//TODO: check whether it is round tick / lot / within min/max notional

	return nil
}

// ValidateBasic is used to quickly disqualify obviously invalid messages quickly
func (msg CancelOrderMsg) ValidateBasic() sdk.Error {
	if len(msg.Sender) == 0 {
		return sdk.ErrUnknownAddress(msg.Sender.String()).TraceSDK("")
	}
	return nil
}
