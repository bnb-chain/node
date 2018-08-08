package rest

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/wire"
)

type order struct {
	BuyQty    utils.Fixed8 `json:"buyQty"`
	BuyPrice  utils.Fixed8 `json:"buyPrice"`
	SellQty   utils.Fixed8 `json:"sellQty"`
	SellPrice utils.Fixed8 `json:"sellPrice"`
}

func decodeOrderBook(cdc *wire.Codec, bz *[]byte) (*[]order, error) {
	table := make([][]int64, 0)
	err := cdc.UnmarshalBinary(*bz, &table)
	if err != nil {
		return nil, err
	}
	book := make([]order, 0)
	for _, o := range table {
		order := order{
			SellQty:   utils.Fixed8(o[0]),
			SellPrice: utils.Fixed8(o[1]),
			BuyPrice:  utils.Fixed8(o[2]),
			BuyQty:    utils.Fixed8(o[3]),
		}
		book = append(book, order)
	}
	return &book, nil
}

func getOrderBook(cdc *wire.Codec, ctx context.CoreContext, pair string) (*[]order, error) {
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
