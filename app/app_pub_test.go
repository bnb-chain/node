package app

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/common/testutils"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
)

// TODO(#66): fix all time.Sleep - potential source of flaky test
func setupAppTest(t *testing.T) (*assert.Assertions, *require.Assertions) {
	logger := log.NewTMLogger(os.Stdout)
	db := dbm.NewMemDB()
	app = NewBinanceChain(logger, db, os.Stdout)
	app.SetEndBlocker(app.EndBlocker)
	app.SetDeliverState(abci.Header{Height: 42, Time: time.Unix(100, 0)})
	app.publicationConfig = &config.PublicationConfig{
		PublishOrderUpdates:   true,
		PublishAccountBalance: true,
		PublishOrderBook:      true,
	}
	app.publisher = pub.NewMockMarketDataPublisher(app.publicationConfig)

	//ctx = app.NewContext(false, abci.Header{ChainID: "mychainid"})
	ctx = app.CheckState.Ctx
	cdc = app.GetCodec()
	keeper = app.DexKeeper
	keeper.CollectOrderInfoForPublish = true
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)
	am = app.AccountKeeper
	_, buyerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000) // give user enough coins to pay the fee
	buyer = buyerAcc.GetAddress()
	_, sellerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000)
	seller = sellerAcc.GetAddress()
	return assert.New(t), require.New(t)
}

func TestAppPub_AddOrder(t *testing.T) {
	assert, require := setupAppTest(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "1", orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 0, 42, 0, 0, ""}, false)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 42})
	time.Sleep(5 * time.Second)

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	require.Len(publisher.BooksPublished, 1)
	require.Len(publisher.BooksPublished[0].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ_BNB", []pub.PriceLevel{{102000, 3000000}}, make([]pub.PriceLevel, 0)}, publisher.BooksPublished[0].Books[0])
}

func TestAppPub_MatchOrder(t *testing.T) {
	assert, require := setupAppTest(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "1", orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 41, 100, 41, 100, 0, ""}, false)
	app.SetDeliverState(abci.Header{Height: 41, Time: time.Unix(100, 0)})
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 41})
	time.Sleep(5 * time.Second)

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	require.Len(publisher.BooksPublished, 1)

	// we add a sell order to fully execute the buyer order
	msg = orderPkg.NewNewOrderMsg(seller, "2", orderPkg.Side.SELL, "XYZ_BNB", 102000, 4000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 101, 42, 101, 0, ""}, false)
	app.SetDeliverState(abci.Header{Height: 42, Time: time.Unix(101, 0)})
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 42})
	time.Sleep(5 * time.Second)

	require.Len(publisher.BooksPublished, 2)
	require.Len(publisher.BooksPublished[1].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ_BNB", []pub.PriceLevel{{102000, 0}}, []pub.PriceLevel{{102000, 1000000}}}, publisher.BooksPublished[1].Books[0])

	// we execute qty 1000000 sell order but add a new qty 1000000 sell order, both buy and sell price level should not publish
	msg = orderPkg.NewNewOrderMsg(buyer, "3", orderPkg.Side.BUY, "XYZ_BNB", 102000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 43, 102, 43, 102, 0, ""}, false)
	msg = orderPkg.NewNewOrderMsg(seller, "4", orderPkg.Side.SELL, "XYZ_BNB", 102000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 43, 102, 43, 102, 0, ""}, false)
	app.SetDeliverState(abci.Header{Height: 43, Time: time.Unix(102, 0)})
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 43})
	time.Sleep(5 * time.Second)

	require.Len(publisher.BooksPublished, 3)
	require.Len(publisher.BooksPublished[2].Books, 0)
}
