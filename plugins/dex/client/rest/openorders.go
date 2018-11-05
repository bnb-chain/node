package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

func OpenOrdersReqHandler(cdc *wire.Codec, ctx context.CLIContext) http.HandlerFunc {
	throw := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.FormValue("symbol")
		addr := r.FormValue("address")

		err := store.ValidatePairSymbol(symbol)
		if err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		}

		// we only verify the addr is legal bech32 address rather than query it from account store
		// we only need make sure the address is acc address rather than validator address because NewOrderMsg only accept acc address
		if len(addr) > 0 && (addr[0] == '"' || addr[0] == '\'') {
			throw(w, http.StatusInternalServerError, fmt.Errorf("addr doesnot need to be wrapped with quotes"))
			return
		}
		if _, err := types.AccAddressFromBech32(addr); err != nil {
			throw(w, http.StatusInternalServerError, fmt.Errorf("addr is not a valid Bech32 address"))
			return
		}
		if openOrders, err := store.GetOpenOrders(cdc, ctx, symbol, addr); err != nil {
			throw(w, http.StatusInternalServerError, err)
			return
		} else {
			err = json.NewEncoder(w).Encode(openOrders)
			if err != nil {
				throw(w, http.StatusInternalServerError, err)
				return
			}
		}

	}
}
