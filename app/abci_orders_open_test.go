package app

import (
	"fmt"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
)

var (
	keeper *orderPkg.Keeper
	buyer  sdk.AccAddress
	seller sdk.AccAddress
	am     auth.AccountMapper
	ctx    sdk.Context
	app    *BinanceChain
	cdc    *wire.Codec
)

func setupOrdersopen(t *testing.T) (*assert.Assertions, *require.Assertions) {
	logger := log.NewTMLogger(os.Stdout)

	db := dbm.NewMemDB()
	app = NewBinanceChain(logger, db, os.Stdout)
	//ctx = app.NewContext(false, abci.Header{ChainID: "mychainid"})
	ctx = app.checkState.ctx
	cdc = app.GetCodec()

	keeper = app.DexKeeper
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	am = app.AccountMapper
	_, buyerAcc := testutils.NewAccount(ctx, am, 100000000000) // give user enough coins to pay the fee
	buyer = buyerAcc.GetAddress()

	_, sellerAcc := testutils.NewAccount(ctx, am, 100000000000)
	seller = sellerAcc.GetAddress()

	return assert.New(t), require.New(t)
}

func Test_Success(t *testing.T) {
	assert, require := setupOrdersopen(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "b-1", orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, 100, 0, 0, ""}, false)

	openOrders := issueQuery(buyer, assert)
	require.Len(openOrders, 1)
	expected := store.OpenOrder{"b-1", "XYZ_BNB", utils.Fixed8(102000), utils.Fixed8(3000000), utils.Fixed8(0), int64(100), int64(0), int64(100), int64(0)}
	assert.Equal(expected, openOrders[0])

	msg = orderPkg.NewNewOrderMsg(seller, "s-1", orderPkg.Side.SELL, "XYZ_BNB", 102000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 101, 1, 101, 1, 0, ""}, false)

	openOrders = issueQuery(seller, assert)
	require.Len(openOrders, 1)
	expected = store.OpenOrder{"s-1", "XYZ_BNB", 102000, 1000000, 0, 101, 1, 101, 1}
	assert.Equal(expected, openOrders[0])

	ctx = ctx.WithBlockHeader(abci.Header{Height: 101, Time: 1})
	ctx = ctx.WithBlockHeight(101)
	keeper.MatchAndAllocateAll(ctx, am, nil)

	openOrders = issueQuery(buyer, assert)
	require.Len(openOrders, 1)
	expected = store.OpenOrder{"b-1", "XYZ_BNB", 102000, 3000000, 1000000, 100, 0, 101, 1}
	assert.Equal(expected, openOrders[0])

	openOrders = issueQuery(seller, assert)
	require.Len(openOrders, 0)

	msg = orderPkg.NewNewOrderMsg(buyer, "b-2", orderPkg.Side.BUY, "XYZ_BNB", 104000, 6000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 102, 2, 102, 2, 0, ""}, false)

	openOrders = issueQuery(buyer, assert)
	require.Len(openOrders, 2)
	require.Contains(openOrders, expected)
	expected = store.OpenOrder{"b-2", "XYZ_BNB", 104000, 6000000, 0, 102, 2, 102, 2}
	require.Contains(openOrders, expected)
}

func Test_NonExistPair(t *testing.T) {
	assert, _ := setupOrdersopen(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "b-1", orderPkg.Side.BUY, "NNB_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, 100, 0, 0, ""}, false)

	openOrders := issueQuery(buyer, assert)
	assert.Empty(openOrders)
}

func Test_NonExistAddr(t *testing.T) {
	assert, _ := setupOrdersopen(t)

	msg := orderPkg.NewNewOrderMsg(seller, "s-1", orderPkg.Side.SELL, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, 100, 0, 0, ""}, false)

	openOrders := issueQuery(buyer, assert)
	assert.Empty(openOrders)
}

func issueQuery(address sdk.AccAddress, assert *assert.Assertions) []store.OpenOrder {
	path := fmt.Sprintf("/%s/openorders/XYZ_BNB/%s", dex.AbciQueryPrefix, address.String())
	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)
	assert.True(sdk.ABCICodeType(res.Code).IsOK())
	openOrders, err := store.DecodeOpenOrders(cdc, &res.Value)
	assert.Nil(err)
	return openOrders
}
