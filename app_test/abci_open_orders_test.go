package app_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
)

func Test_Success(t *testing.T) {
	assert, require, pair := setup(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "b-1", orderPkg.Side.BUY, pair, 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, 100, 0, 0, "", 0}, false)

	openOrders := issueMustSuccessQuery(pair, buyer, assert)
	require.Len(openOrders, 1)
	expected := store.OpenOrder{"b-1", pair, utils.Fixed8(102000), utils.Fixed8(3000000), utils.Fixed8(0), int64(100), int64(0), int64(100), int64(0)}
	assert.Equal(expected, openOrders[0])

	msg = orderPkg.NewNewOrderMsg(seller, "s-1", orderPkg.Side.SELL, pair, 102000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 101, 1, 101, 1, 0, "", 0}, false)

	openOrders = issueMustSuccessQuery(pair, seller, assert)
	require.Len(openOrders, 1)
	expected = store.OpenOrder{"s-1", pair, 102000, 1000000, 0, 101, 1, 101, 1}
	assert.Equal(expected, openOrders[0])

	ctx = ctx.WithBlockHeader(abci.Header{Height: 101, Time: time.Unix(1, 0)})
	ctx = ctx.WithBlockHeight(101)
	keeper.MatchAndAllocateAll(ctx, nil, false)

	openOrders = issueMustSuccessQuery(pair, buyer, assert)
	require.Len(openOrders, 1)
	expected = store.OpenOrder{"b-1", pair, 102000, 3000000, 1000000, 100, 0, 101, 1000000000}
	assert.Equal(expected, openOrders[0])

	openOrders = issueMustSuccessQuery(pair, seller, assert)
	require.Len(openOrders, 0)

	msg = orderPkg.NewNewOrderMsg(buyer, "b-2", orderPkg.Side.BUY, pair, 104000, 6000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 102, 2, 102, 2, 0, "", 0}, false)

	openOrders = issueMustSuccessQuery(pair, buyer, assert)
	require.Len(openOrders, 2)
	require.Contains(openOrders, expected)
	expected = store.OpenOrder{"b-2", pair, 104000, 6000000, 0, 102, 2, 102, 2}
	require.Contains(openOrders, expected)
}

func Test_InvalidPair(t *testing.T) {
	assert, _, _ := setup(t)

	res := issueQuery("%afuiewf%@^&2blf", buyer.String())
	assert.Equal(uint32(sdk.CodeInternal), res.Code)
	assert.Equal("pair is not valid", res.Log)
}

func Test_NonListedPair(t *testing.T) {
	assert, _, _ := setup(t)

	res := issueQuery("NNB-000_BNB", buyer.String())
	assert.Equal(uint32(sdk.CodeInternal), res.Code)
	assert.Equal("pair is not listed", res.Log)
}

func Test_InvalidAddr(t *testing.T) {
	assert, _, pair := setup(t)

	res := issueQuery(pair, "%afuiewf%@^&2blf")
	assert.Equal(uint32(sdk.CodeInternal), res.Code)
	assert.Equal("address is not valid", res.Log)
}

func Test_NonExistAddr(t *testing.T) {
	assert, _, pair := setup(t)

	msg := orderPkg.NewNewOrderMsg(seller, "s-1", orderPkg.Side.SELL, pair, 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, 100, 0, 0, "", 0}, false)

	openOrders := issueMustSuccessQuery(pair, buyer, assert)
	assert.Empty(openOrders)
}

func issueMustSuccessQuery(pair string, address sdk.AccAddress, assert *assert.Assertions) []store.OpenOrder {
	res := issueQuery(pair, address.String())
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
