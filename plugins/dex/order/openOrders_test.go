package order

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/plugins/dex/types"
)

// mainly used to test keeper.GetOpenOrders API

const (
	ZzAddr = "cosmosaccaddr17pu78tfxdmd0wmkcl8papvt3zxrpmpkza5809m"
	ZcAddr = "cosmosaccaddr194epkcnk0aganvjnwpj47nfztjl2ur9wujpj6h"
)

var (
	zz, _ = sdk.AccAddressFromBech32(ZzAddr)
	zc, _ = sdk.AccAddressFromBech32(ZcAddr)
)

func initKeeper() *Keeper {
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
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
