package app

import (
	"os"
	"testing"
	"time"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/common/testutils"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
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

// TODO(#66): fix all time.Sleep - potential source of flaky test
func setupAppTest(t *testing.T) (*assert.Assertions, *require.Assertions) {
	logger := log.NewTMLogger(os.Stdout)
	db := dbm.NewMemDB()
	app = NewBinanceChain(logger, db, os.Stdout)
	app.SetEndBlocker(app.EndBlocker)
	app.setDeliverState(abci.Header{Height: 42, Time: 100})
	app.publicationConfig = &config.PublicationConfig{
		PublishOrderUpdates:   true,
		PublishAccountBalance: true,
		PublishOrderBook:      true,
	}
	app.publisher = pub.NewMockMarketDataPublisher(app.publicationConfig)

	//ctx = app.NewContext(false, abci.Header{ChainID: "mychainid"})
	ctx = app.checkState.ctx
	cdc = app.GetCodec()
	keeper = app.DexKeeper
	keeper.CollectOrderInfoForPublish = true
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

func TestAppPub_AddOrder(t *testing.T) {
	assert, require := setupAppTest(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "1", orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, ""}, 42, false)
	app.EndBlocker(ctx, abci.RequestEndBlock{42})
	time.Sleep(5 * time.Second)

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	require.Len(publisher.BooksPublished, 1)
	require.Len(publisher.BooksPublished[0].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ_BNB", []pub.PriceLevel{{102000, 3000000}}, make([]pub.PriceLevel, 0)}, publisher.BooksPublished[0].Books[0])
}

func TestAppPub_MatchOrder(t *testing.T) {
	assert, require := setupAppTest(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "1", orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 100, 0, ""}, 41, false)
	app.setDeliverState(abci.Header{Height: 41, Time: 100})
	app.EndBlocker(ctx, abci.RequestEndBlock{41})
	time.Sleep(5 * time.Second)

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	require.Len(publisher.BooksPublished, 1)

	// we add a sell order to fully execute the buyer order
	msg = orderPkg.NewNewOrderMsg(seller, "2", orderPkg.Side.SELL, "XYZ_BNB", 102000, 4000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 101, 0, ""}, 42, false)
	app.setDeliverState(abci.Header{Height: 42, Time: 101})
	app.endBlocker(ctx, abci.RequestEndBlock{42})
	time.Sleep(5 * time.Second)

	require.Len(publisher.BooksPublished, 2)
	require.Len(publisher.BooksPublished[1].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ_BNB", []pub.PriceLevel{{102000, 0}}, []pub.PriceLevel{{102000, 1000000}}}, publisher.BooksPublished[1].Books[0])

	// we execute qty 1000000 sell order but add a new qty 1000000 sell order, both buy and sell price level should not publish
	msg = orderPkg.NewNewOrderMsg(buyer, "3", orderPkg.Side.BUY, "XYZ_BNB", 102000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 102, 0, ""}, 43, false)
	msg = orderPkg.NewNewOrderMsg(seller, "4", orderPkg.Side.SELL, "XYZ_BNB", 102000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 102, 0, ""}, 43, false)
	app.setDeliverState(abci.Header{Height: 43, Time: 102})
	app.endBlocker(ctx, abci.RequestEndBlock{43})
	time.Sleep(5 * time.Second)

	require.Len(publisher.BooksPublished, 3)
	require.Len(publisher.BooksPublished[2].Books, 0)
}
