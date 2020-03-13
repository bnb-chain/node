package order

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/dex/types"
)

// mainly used to test keeper.GetOpenOrders API

const (
	ZzAddr = "cosmos1a4y3tjwzgemg0g05fl8ucea0ftkj28l3cfes6q"
	ZcAddr = "cosmos1al5dssf3g6xjmjykd2e36pxprq6jh6y24j9ers"
)

var (
	zz, _ = sdk.AccAddressFromBech32(ZzAddr)
	zc, _ = sdk.AccAddressFromBech32(ZcAddr)
)

func initKeeper() *Keeper {
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc, false)
	return keeper
}

func TestOpenOrders_NoSymbol(t *testing.T) {
	keeper := initKeeper()

	res := keeper.GetOpenOrders("NNB_BNB", zz)
	if len(res) == 0 {
		t.Log("Get expected empty result for a non-existing pair")
	}
}

func TestOpenOrders_NoAddr(t *testing.T) {
	keeper := initKeeper()

	keeper.AddEngine(types.NewTradingPair("NNB", "BNB", 100000000))
	res := keeper.GetOpenOrders("NNB_BNB", zz)
	if len(res) == 0 {
		t.Log("Get expected empty result for a non-existing addr")
	}
}
