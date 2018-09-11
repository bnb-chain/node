package store

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/wire"
)

// queryOrderBook queries the store for the serialized order book for a given pair.
func queryOrderBook(ctx context.CLIContext, pair string) (*[]byte, error) {
	bz, err := ctx.Query(fmt.Sprintf("app/orderbook/%s", pair))
	if err != nil {
		return nil, err
	}
	return &bz, nil
}

// decodeOrderBook decodes the order book to a set of OrderBookLevel structs
func decodeOrderBook(cdc *wire.Codec, bz *[]byte) (*[]OrderBookLevel, error) {
	levels := make([]OrderBookLevel, 0)
	err := cdc.UnmarshalBinary(*bz, &levels)
	if err != nil {
		return nil, err
	}
	return &levels, nil
}

// GetOrderBook decodes the order book from the serialized store
func GetOrderBook(cdc *wire.Codec, ctx context.CLIContext, pair string) (*[]OrderBookLevel, error) {
	bz, err := queryOrderBook(ctx, pair)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, nil
	}
	book, err := decodeOrderBook(cdc, bz)
	return book, err
}
