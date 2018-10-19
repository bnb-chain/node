package order

import (
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"testing"
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

func TestOpenOrders_AfterMatch(t *testing.T) {
	assert := assert.New(t)
	keeper := initKeeper()
	keeper.AddEngine(types.NewTradingPair("NNB", "BNB", 100000000))

	// add an original buy order, waiting to be filled
	msg := NewNewOrderMsg(zc, ZcAddr+"-0", Side.BUY, "NNB_BNB", 1000000000, 1000000000)
	orderInfo := OrderInfo{msg, 42, 84, 42, 84, 0, ""}
	keeper.AddOrder(orderInfo, false)
	res := keeper.GetOpenOrders("NNB_BNB", zc)
	assert.Equal(1, len(res))
	assert.Equal("NNB_BNB", res[0].Symbol)
	assert.Equal(ZcAddr+"-0", res[0].Id)
	assert.Equal(utils.Fixed8(0), res[0].CumQty)
	assert.Equal(utils.Fixed8(1000000000), res[0].Price)
	assert.Equal(utils.Fixed8(1000000000), res[0].Quantity)
	assert.Equal(int64(42), res[0].CreatedHeight)
	assert.Equal(int64(84), res[0].CreatedTimestamp)
	assert.Equal(int64(42), res[0].LastUpdatedHeight)
	assert.Equal(int64(84), res[0].LastUpdatedTimestamp)

	// add a sell order, partialled fill the buy order
	msg = NewNewOrderMsg(zz, ZzAddr+"-0", Side.SELL, "NNB_BNB", 900000000, 300000000)
	orderInfo = OrderInfo{msg, 43, 86, 43, 86, 0, ""}
	keeper.AddOrder(orderInfo, false)
	res = keeper.GetOpenOrders("NNB_BNB", zz)
	assert.Equal(1, len(res))

	// match existing two orders
	matchRes, _ := keeper.MatchAll(43, 86)
	assert.Equal(sdk.CodeOK, matchRes)

	// after match, the original buy order's cumQty and latest updated fields should be updated
	res = keeper.GetOpenOrders("NNB_BNB", zc)
	assert.Equal(1, len(res))
	assert.Equal(utils.Fixed8(300000000), res[0].CumQty)
	assert.Equal(utils.Fixed8(1000000000), res[0].Price)    // price shouldn't change
	assert.Equal(utils.Fixed8(1000000000), res[0].Quantity) // quantity shouldn't change
	assert.Equal(int64(42), res[0].CreatedHeight)
	assert.Equal(int64(84), res[0].CreatedTimestamp)
	assert.Equal(int64(43), res[0].LastUpdatedHeight)
	assert.Equal(int64(86), res[0].LastUpdatedTimestamp)

	// after match, the sell order should be closed
	res = keeper.GetOpenOrders("NNB_BNB", zz)
	assert.Equal(0, len(res))

	// add another sell order to fully fill original buy order
	msg = NewNewOrderMsg(zz, ZzAddr+"-1", Side.SELL, "NNB_BNB", 1000000000, 700000000)
	orderInfo = OrderInfo{msg, 44, 88, 44, 88, 0, ""}
	keeper.AddOrder(orderInfo, false)
	res = keeper.GetOpenOrders("NNB_BNB", zz)
	assert.Equal(1, len(res))
	assert.Equal("NNB_BNB", res[0].Symbol)
	assert.Equal(ZzAddr+"-1", res[0].Id)
	assert.Equal(utils.Fixed8(0), res[0].CumQty)
	assert.Equal(utils.Fixed8(1000000000), res[0].Price)
	assert.Equal(utils.Fixed8(700000000), res[0].Quantity)
	assert.Equal(int64(44), res[0].CreatedHeight)
	assert.Equal(int64(88), res[0].CreatedTimestamp)
	assert.Equal(int64(44), res[0].LastUpdatedHeight)
	assert.Equal(int64(88), res[0].LastUpdatedTimestamp)

	// match existing two orders
	matchRes, _ = keeper.MatchAll(44, 88)
	assert.Equal(sdk.CodeOK, matchRes)

	// after match, all orders should be closed
	res = keeper.GetOpenOrders("NNB_BNB", zc)
	assert.Equal(0, len(res))
	res = keeper.GetOpenOrders("NNB_BNB", zz)
	assert.Equal(0, len(res))
}
