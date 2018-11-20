package app_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
)

func Test_Success(t *testing.T) {
	assert, require := setup(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "b-1", orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, 100, 0, 0, ""}, false)

	openOrders := issueMustSuccessQuery(buyer, assert)
	require.Len(openOrders, 1)
	expected := store.OpenOrder{"b-1", "XYZ_BNB", utils.Fixed8(102000), utils.Fixed8(3000000), utils.Fixed8(0), int64(100), int64(0), int64(100), int64(0)}
	assert.Equal(expected, openOrders[0])

	msg = orderPkg.NewNewOrderMsg(seller, "s-1", orderPkg.Side.SELL, "XYZ_BNB", 102000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 101, 1, 101, 1, 0, ""}, false)

	openOrders = issueMustSuccessQuery(seller, assert)
	require.Len(openOrders, 1)
	expected = store.OpenOrder{"s-1", "XYZ_BNB", 102000, 1000000, 0, 101, 1, 101, 1}
	assert.Equal(expected, openOrders[0])

	ctx = ctx.WithBlockHeader(abci.Header{Height: 101, Time: time.Unix(1, 0)})
	ctx = ctx.WithBlockHeight(101)
	keeper.MatchAndAllocateAll(ctx, nil)

	openOrders = issueMustSuccessQuery(buyer, assert)
	require.Len(openOrders, 1)
	expected = store.OpenOrder{"b-1", "XYZ_BNB", 102000, 3000000, 1000000, 100, 0, 101, 1}
	assert.Equal(expected, openOrders[0])

	openOrders = issueMustSuccessQuery(seller, assert)
	require.Len(openOrders, 0)

	msg = orderPkg.NewNewOrderMsg(buyer, "b-2", orderPkg.Side.BUY, "XYZ_BNB", 104000, 6000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 102, 2, 102, 2, 0, ""}, false)

	openOrders = issueMustSuccessQuery(buyer, assert)
	require.Len(openOrders, 2)
	require.Contains(openOrders, expected)
	expected = store.OpenOrder{"b-2", "XYZ_BNB", 104000, 6000000, 0, 102, 2, 102, 2}
	require.Contains(openOrders, expected)
}

func Test_InvalidPair(t *testing.T) {
	assert, _ := setup(t)

	res := issueQuery("%afuiewf%@^&2blf", buyer.String())
	assert.Equal(uint32(sdk.CodeInternal), res.Code)
	assert.Equal("pair is not valid", res.Log)
}

func Test_NonListedPair(t *testing.T) {
	assert, _ := setup(t)

	res := issueQuery("NNB_BNB", buyer.String())
	assert.Equal(uint32(sdk.CodeInternal), res.Code)
	assert.Equal("pair is not listed", res.Log)
}

func Test_InvalidAddr(t *testing.T) {
	assert, _ := setup(t)

	res := issueQuery("XYZ_BNB", "%afuiewf%@^&2blf")
	assert.Equal(uint32(sdk.CodeInternal), res.Code)
	assert.Equal("address is not valid", res.Log)
}

func Test_NonExistAddr(t *testing.T) {
	assert, _ := setup(t)

	msg := orderPkg.NewNewOrderMsg(seller, "s-1", orderPkg.Side.SELL, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, 100, 0, 0, ""}, false)

	openOrders := issueMustSuccessQuery(buyer, assert)
	assert.Empty(openOrders)
}

func issueMustSuccessQuery(address sdk.AccAddress, assert *assert.Assertions) []store.OpenOrder {
	res := issueQuery("XYZ_BNB", address.String())
	assert.True(sdk.ABCICodeType(res.Code).IsOK())
	openOrders, err := store.DecodeOpenOrders(cdc, &res.Value)
	assert.Nil(err)
	return openOrders
}

func issueQuery(pair string, address string) abci.ResponseQuery {
	path := fmt.Sprintf("/%s/openorders/%s/%s", dex.AbciQueryPrefix, pair, address)
	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	return app.Query(query)
}
