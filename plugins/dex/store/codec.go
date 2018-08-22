package store

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/wire"
)

func queryOrderBook(cdc *wire.Codec, ctx context.CoreContext, pair string) ([]byte, error) {
	bz, err := ctx.Query(fmt.Sprintf("app/orderbook/%s", pair))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// GetOrderBook decodes the order book from the serialized store
func GetOrderBook(cdc *wire.Codec, ctx context.CoreContext, pair string) (*[]OrderBookLevel, error) {
	bz, err := queryOrderBook(cdc, ctx, pair)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, nil
	}
	book, err := DecodeOrderBook(cdc, &bz)
	return book, err
}

// DecodeOrderBook decodes the order book to a set of OrderBookLevel structs
func DecodeOrderBook(cdc *wire.Codec, bz *[]byte) (*[]OrderBookLevel, error) {
	levels := make([]OrderBookLevel, 0)
	err := cdc.UnmarshalBinary(*bz, &levels)
	if err != nil {
		return nil, err
	}
	return &levels, nil
}
