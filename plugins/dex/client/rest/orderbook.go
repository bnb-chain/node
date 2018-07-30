package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/wire"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
)

type order struct {
	BuyQty    utils.Fixed8 `json:"buyQty"`
	BuyPrice  utils.Fixed8 `json:"buyPrice"`
	SellQty   utils.Fixed8 `json:"sellQty"`
	SellPrice utils.Fixed8 `json:"sellPrice"`
}

type booksResponse struct {
	Pair   string  `json:"pair"`
	Orders []order `json:"orders"`
}

func registerBooksRoute(
	ctx context.CoreContext,
	r *mux.Router,
	cdc *wire.Codec,
) {
	r.HandleFunc("/orderbook/{pair}", BooksRequestHandler(cdc, ctx)).Methods("GET")
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

// BooksRequestHandler - http request handler to send coins to a address
func BooksRequestHandler(
	cdc *wire.Codec, ctx context.CoreContext,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		throw := func(status int, err error) {
			w.WriteHeader(status)
			w.Write([]byte(err.Error()))
			return
		}

		vars := mux.Vars(r)

		// collect params
		params := struct {
			pair string
		}{
			pair: vars["pair"],
		}

		// validate pair
		err := validatePairSymbol(params.pair)
		if err != nil {
			throw(http.StatusNotFound, err)
			return
		}

		book, err := getOrderBook(cdc, ctx, params.pair)
		if err != nil {
			throw(http.StatusNotFound, err)
			return
		}

		resp := booksResponse{
			Pair:   vars["pair"],
			Orders: *book,
		}

		output, err := cdc.MarshalJSON(resp)
		if err != nil {
			throw(http.StatusInternalServerError, err)
			return
		}

		w.Write(output)
	}
}

func validatePairSymbol(symbol string) error {
	tokenSymbols := strings.Split(symbol, "_")
	if len(tokenSymbols) != 2 {
		return errors.New("Invalid symbol")
	}

	for _, tokenSymbol := range tokenSymbols {
		err := types.ValidateSymbol(tokenSymbol)
		if err != nil {
			return err
		}
	}

	return nil
}
