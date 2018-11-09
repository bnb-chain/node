package utils

import (
	"errors"
	"fmt"
	"io"

	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
)

func write(w io.Writer, data string) error {
	if _, err := w.Write([]byte(data)); err != nil {
		return err
	}
	return nil
}

// StreamDepthResponse streams out the order book in the http response.
func StreamDepthResponse(w io.Writer, ob *store.OrderBook, limit int) error {
	// assuming MaxDepthLevels is used in caller, which it should be
	if dex.MaxDepthLevels < limit {
		return errors.New("StreamDepthResponse: MaxDepthLevels greater than limit. Unable to stream up to limit")
	}

	levels := ob.Levels

	// output must be equivalent to SortJSON output (for tests)
	preamble := "{\"asks\":["
	if err := write(w, preamble); err != nil {
		return err
	}

	// pass 1 - asks
	i := 0
	for _, o := range levels {
		if i > limit-1 {
			break
		}
		// skip zero qty level
		if o.SellQty == 0 {
			continue
		}
		if i > 0 {
			if err := write(w, ","); err != nil {
				return err
			}
		}
		// [PRICE, QTY]
		if err := write(w, fmt.Sprintf("[\"%s\",\"%s\"]", o.SellPrice, o.SellQty)); err != nil {
			return err
		}
		i++
	}

	// pass 2 - bids
	if err := write(w, "],\"bids\":["); err != nil {
		return err
	}
	i = 0
	for _, o := range levels {
		if i > limit-1 {
			break
		}
		// skip zero qty level
		if o.BuyQty == 0 {
			continue
		}
		if i > 0 {
			if err := write(w, ","); err != nil {
				return err
			}
		}
		// [PRICE, QTY]
		if err := write(w, fmt.Sprintf("[\"%s\",\"%s\"]", o.BuyPrice, o.BuyQty)); err != nil {
			return err
		}
		i++
	}

	// end streamed json with height
	return write(w, fmt.Sprintf("],\"height\":%d}", ob.Height))
}
