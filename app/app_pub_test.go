package app

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/common/testutils"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
)

const (
	expireFee    = 1000
	iocExpireFee = 500
)

// TODO(#66): fix all time.Sleep - potential source of flaky test
func setupAppTest(t *testing.T) (*assert.Assertions, *require.Assertions) {
	logger := log.NewTMLogger(os.Stdout)
	db := dbm.NewMemDB()
	app = NewBinanceChain(logger, db, os.Stdout)
	app.SetEndBlocker(app.EndBlocker)
	am = app.AccountKeeper
	ctx = sdk.NewContext(app.GetCommitMultiStore(), abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	_, proposerAcc := testutils.NewAccount(ctx, am, 100)
	proposerValAddr := ed25519.GenPrivKey().PubKey().Address()
	app.ValAddrMapper.SetVal(ctx, proposerValAddr, proposerAcc.GetAddress())
	proposer := abci.Validator{Address: proposerValAddr, Power: 10}
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: proposerValAddr}).WithVoteInfos([]abci.VoteInfo{
		{Validator: proposer, SignedLastBlock: true},
	})

	app.SetDeliverState(abci.Header{Height: 42, Time: time.Unix(0, 100), ProposerAddress: proposerValAddr})
	app.SetCheckState(abci.Header{Height: 42, Time: time.Unix(0, 100), ProposerAddress: proposerValAddr})
	app.publicationConfig = &config.PublicationConfig{
		PublishOrderUpdates:   true,
		PublishAccountBalance: true,
		PublishOrderBook:      true,
	}
	app.publisher = pub.NewMockMarketDataPublisher(app.publicationConfig)

	//ctx = app.NewContext(false, abci.Header{ChainID: "mychainid"})
	ctx = app.DeliverState.Ctx
	cdc = app.GetCodec()
	keeper = app.DexKeeper
	keeper.CollectOrderInfoForPublish = true
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)
	keeper.FeeManager.FeeConfig.ExpireFee = expireFee
	keeper.FeeManager.FeeConfig.IOCExpireFee = iocExpireFee
	keeper.FeeManager.FeeConfig.FeeRate = 1000
	keeper.FeeManager.FeeConfig.FeeRateNative = 500

	_, buyerAcc = testutils.NewAccountForPub(ctx, am, 100000000000, 0, 0) // give user enough coins to pay the fee
	buyer = buyerAcc.GetAddress()
	_, sellerAcc = testutils.NewAccountForPub(ctx, am, 100000000000, 0, 0)
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

	msg := orderPkg.NewNewOrderMsg(buyer, orderPkg.GenerateOrderID(1, buyer), orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	app.SetDeliverState(abci.Header{Height: 41, Time: time.Unix(0, 100)})
	handler := orderPkg.NewHandler(cdc, keeper, am)
	buyerAcc.SetSequence(1)
	am.SetAccount(ctx, buyerAcc)
	ctx = ctx.WithValue(baseapp.TxHashKey, "")
	res := handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 41})
	time.Sleep(5 * time.Second)

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	require.Len(publisher.BooksPublished, 1)
	require.Len(publisher.AccountPublished, 1)
	require.Len(publisher.AccountPublished[0].Accounts, 1)
	expectedAccountToPub := pub.Account{buyer.String(), []*pub.AssetBalance{{"BNB", 99999996940, 0, 3060}, {"XYZ", 100000000000, 0, 0}}}
	require.Equal(expectedAccountToPub, publisher.AccountPublished[0].Accounts[0])

	// we add a sell order to fully execute the buyer order
	msg = orderPkg.NewNewOrderMsg(seller, orderPkg.GenerateOrderID(1, seller), orderPkg.Side.SELL, "XYZ_BNB", 102000, 4000000)
	app.SetDeliverState(abci.Header{Height: 42, Time: time.Unix(0, 101)})
	sellerAcc.SetSequence(1)
	am.SetAccount(ctx, sellerAcc)
	res = handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 42})
	time.Sleep(5 * time.Second)

	require.Len(publisher.BooksPublished, 2)
	require.Len(publisher.BooksPublished[1].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ_BNB", []pub.PriceLevel{{102000, 0}}, []pub.PriceLevel{{102000, 1000000}}}, publisher.BooksPublished[1].Books[0])
	expectedAccountToPub = pub.Account{buyer.String(), []*pub.AssetBalance{{"BNB", 99999995440, 0, 0}, {"XYZ", 100003000000, 0, 0}}}
	expectedAccountToPubSeller := pub.Account{seller.String(), []*pub.AssetBalance{{"BNB", 100000003059, 0, 0}, {"XYZ", 99996000000, 0, 1000000}}}
	require.Len(publisher.AccountPublished, 2)
	require.Len(publisher.AccountPublished[1].Accounts, 2)
	require.Contains(publisher.AccountPublished[1].Accounts, expectedAccountToPub)
	require.Contains(publisher.AccountPublished[1].Accounts, expectedAccountToPubSeller)

	// we execute qty 1000000 sell order but add a new qty 1000000 sell order, both buy and sell price level should not publish
	msg = orderPkg.NewNewOrderMsg(buyer, orderPkg.GenerateOrderID(2, buyer), orderPkg.Side.BUY, "XYZ_BNB", 102000, 1000000)
	app.SetDeliverState(abci.Header{Height: 43, Time: time.Unix(0, 102)})
	buyerAcc.SetSequence(2)
	am.SetAccount(ctx, buyerAcc)
	res = handler(ctx, msg)
	msg = orderPkg.NewNewOrderMsg(seller, orderPkg.GenerateOrderID(2, seller), orderPkg.Side.SELL, "XYZ_BNB", 102000, 1000000)
	sellerAcc.SetSequence(2)
	am.SetAccount(ctx, sellerAcc)
	res = handler(ctx, msg)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 43})
	time.Sleep(5 * time.Second)
	expectedAccountToPub = pub.Account{buyer.String(), []*pub.AssetBalance{{"BNB", 99999998980, 0, 0}, {"XYZ", 100001000000, 0, 0}}}
	expectedAccountToPubSeller = pub.Account{seller.String(), []*pub.AssetBalance{{"BNB", 100000001020, 0, 0}, {"XYZ", 99999000000, 0, 0}}}

	require.Len(publisher.BooksPublished, 3)
	require.Len(publisher.BooksPublished[2].Books, 0)
	require.Len(publisher.AccountPublished, 3)
	require.Len(publisher.AccountPublished[2].Accounts, 2)
	require.Contains(publisher.AccountPublished[2].Accounts, expectedAccountToPub)
	require.Contains(publisher.AccountPublished[2].Accounts, expectedAccountToPubSeller)
}
