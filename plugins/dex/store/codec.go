package store

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/wire"
)

// GetOrderBook decodes the OrderBook from the serialization
func GetOrderBook(cdc *wire.Codec, ctx context.CoreContext, pair string) (*[]OrderBookLevel, error) {
	bz, err := ctx.Query(fmt.Sprintf("app/orderbook/%s", pair))
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, nil
	}
	book, err := decodeOrderBook(cdc, &bz)
	return book, err
}

func decodeOrderBook(cdc *wire.Codec, bz *[]byte) (*[]OrderBookLevel, error) {
	table := make([][]int64, 0)
	err := cdc.UnmarshalBinary(*bz, &table)
	if err != nil {
		return nil, err
	}
	book := make([]OrderBookLevel, 0)
	for _, o := range table {
		order := OrderBookLevel{
			SellQty:   utils.Fixed8(o[0]),
			SellPrice: utils.Fixed8(o[1]),
			BuyPrice:  utils.Fixed8(o[2]),
			BuyQty:    utils.Fixed8(o[3]),
		}
		book = append(book, order)
	}
	return &book, nil
}
