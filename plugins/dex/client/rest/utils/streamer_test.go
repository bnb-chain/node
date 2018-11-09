package utils_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"

	util "github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/dex/client/rest/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
)

func TestStreamDepthResponse(t *testing.T) {
	type args struct {
		ob    *store.OrderBook
		limit int
	}
	type response struct {
		Height int64      `json:"height"`
		Asks   [][]string `json:"asks"`
		Bids   [][]string `json:"bids"`
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		// happy
		{
			name: "Empty order book",
			args: args{
				ob: &store.OrderBook{
					Height: 1,
					Levels: []store.OrderBookLevel{},
				},
				limit: 50,
			},
			wantW: (func() string {
				want, _ := json.Marshal(response{
					Height: 1,
					Asks:   [][]string{},
					Bids:   [][]string{},
				})
				sorted := types.MustSortJSON(want)
				return string(sorted)
			})(),
			wantErr: false,
		},
		{
			name: "Normal order book",
			args: args{
				ob: &store.OrderBook{
					Height: 1,
					Levels: []store.OrderBookLevel{
						{
							BuyQty:    util.NewFixed8(1),
							BuyPrice:  util.NewFixed8(100),
							SellQty:   util.NewFixed8(1),
							SellPrice: util.NewFixed8(200),
						},
						{
							BuyQty:    util.NewFixed8(0),
							BuyPrice:  util.NewFixed8(0),
							SellQty:   util.NewFixed8(1),
							SellPrice: util.NewFixed8(300),
						},
					},
				},
				limit: 50,
			},
			wantW: (func() string {
				want, _ := json.Marshal(response{
					Height: 1,
					Asks: [][]string{
						{"200.00000000", "1.00000000"},
						{"300.00000000", "1.00000000"},
					},
					Bids: [][]string{
						{"100.00000000", "1.00000000"},
					},
				})
				sorted := types.MustSortJSON(want)
				return string(sorted)
			})(),
			wantErr: false,
		},
		{
			name: "Order book with zero quantity levels",
			args: args{
				ob: &store.OrderBook{
					Height: 1,
					Levels: []store.OrderBookLevel{
						{
							BuyQty:    util.NewFixed8(1),
							BuyPrice:  util.NewFixed8(150),
							SellQty:   util.NewFixed8(1),
							SellPrice: util.NewFixed8(200),
						},
						{
							BuyQty:    util.NewFixed8(0),
							BuyPrice:  util.NewFixed8(0),
							SellQty:   util.NewFixed8(0),
							SellPrice: util.NewFixed8(0),
						},
						{
							BuyQty:    util.NewFixed8(1),
							BuyPrice:  util.NewFixed8(100),
							SellQty:   util.NewFixed8(0),
							SellPrice: util.NewFixed8(300),
						},
					},
				},
				limit: 50,
			},
			wantW: (func() string {
				want, _ := json.Marshal(response{
					Height: 1,
					Asks: [][]string{
						{"200.00000000", "1.00000000"},
					},
					Bids: [][]string{
						{"150.00000000", "1.00000000"},
						{"100.00000000", "1.00000000"},
					},
				})
				sorted := types.MustSortJSON(want)
				return string(sorted)
			})(),
			wantErr: false,
		},
		{
			name: "Order book with more levels than limit",
			args: args{
				ob: &store.OrderBook{
					Height: 1,
					Levels: []store.OrderBookLevel{
						{
							BuyQty:    util.NewFixed8(0),
							BuyPrice:  util.NewFixed8(0),
							SellQty:   util.NewFixed8(1),
							SellPrice: util.NewFixed8(200),
						},
						{
							BuyQty:    util.NewFixed8(1),
							BuyPrice:  util.NewFixed8(100),
							SellQty:   util.NewFixed8(1),
							SellPrice: util.NewFixed8(300),
						},
					},
				},
				limit: 1,
			},
			wantW: (func() string {
				want, _ := json.Marshal(response{
					Height: 1,
					Asks: [][]string{
						// cheapest sell first
						{"200.00000000", "1.00000000"},
					},
					Bids: [][]string{
						// highest buy first
						{"100.00000000", "1.00000000"},
					},
				})
				sorted := types.MustSortJSON(want)
				return string(sorted)
			})(),
			wantErr: false,
		},
		// errors
		{
			name: "Error: Limit is greater than MaxDepthLevels",
			args: args{
				ob: &store.OrderBook{
					Height: 1,
					Levels: []store.OrderBookLevel{},
				},
				limit: dex.MaxDepthLevels + 1,
			},
			wantW:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := utils.StreamDepthResponse(w, tt.args.ob, tt.args.limit); (err != nil) != tt.wantErr {
				t.Errorf("StreamDepthResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("StreamDepthResponse() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
