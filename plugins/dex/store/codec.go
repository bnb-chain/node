package store

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/wire"
)

func queryOrderBook(cdc *wire.Codec, ctx context.CoreContext, pair string) ([]byte, error) {
	bz, err := ctx.Query(fmt.Sprintf("app/orderbook/%s", pair))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// GetOrderBook decodes the order book from the store
func GetOrderBook(cdc *wire.Codec, ctx context.CoreContext, pair string) (*[]Order, error) {
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

// GetOrderBookRaw decodes the raw order book from the store
func GetOrderBookRaw(cdc *wire.Codec, ctx context.CoreContext, pair string) (*[][]int64, error) {
	bz, err := queryOrderBook(cdc, ctx, pair)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, nil
	}
	table, err := DecodeOrderBookRaw(cdc, &bz)
	return table, err
}

// DecodeOrderBook decodes the order book to a set of Order structs
func DecodeOrderBook(cdc *wire.Codec, bz *[]byte) (*[]Order, error) {
	table, err := DecodeOrderBookRaw(cdc, bz)
	if err != nil {
		return nil, err
	}
	book := make([]Order, 0)
	for _, o := range *table {
		order := Order{
			SellQty:   utils.Fixed8(o[0]),
			SellPrice: utils.Fixed8(o[1]),
			BuyPrice:  utils.Fixed8(o[2]),
			BuyQty:    utils.Fixed8(o[3]),
		}
		book = append(book, order)
	}
	return &book, nil
}

// DecodeOrderBookRaw decodes the raw order book table
func DecodeOrderBookRaw(cdc *wire.Codec, bz *[]byte) (*[][]int64, error) {
	table := make([][]int64, 0)
	err := cdc.UnmarshalBinary(*bz, &table)
	if err != nil {
		return nil, err
	}
	return &table, nil
}
