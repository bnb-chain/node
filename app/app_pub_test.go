package app

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/common/testutils"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

const (
	expireFee    = 1000
	iocExpireFee = 500
)

func prepareGenTx(cdc *codec.Codec, chainId string,
	valOperAddr sdk.ValAddress, valPubKey crypto.PubKey) json.RawMessage {
	msg := stake.NewMsgCreateValidator(
		valOperAddr,
		valPubKey,
		DefaultSelfDelegationToken,
		stake.NewDescription("pub", "", "", ""),
		stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
	)
	tx := auth.NewStdTx([]sdk.Msg{msg}, nil, "")
	txBytes, err := wire.MarshalJSONIndent(cdc, tx)
	if err != nil {
		panic(err)
	}

	return txBytes
}

func setupAppTest(t *testing.T) (*assert.Assertions, *require.Assertions) {
	logger := log.NewNopLogger()
	db := dbm.NewMemDB()

	app = NewBinanceChain(logger, db, os.Stdout)
	app.SetAnteHandler(nil)
	app.SetDeliverState(abci.Header{})
	am = app.AccountKeeper
	ctx = app.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	_, proposerAcc := testutils.NewAccount(ctx, am, 100)
	proposerPubKey := ed25519.GenPrivKey().PubKey()
	proposerValAddr := proposerPubKey.Address()
	app.ValAddrMapper.SetVal(ctx, proposerValAddr, proposerAcc.GetAddress())
	// set ante handler to nil to skip the sig verification. the side effect is that we also skip the tx fee collection.
	chainId := "chain-pub"
	genTx := prepareGenTx(app.Codec, chainId, sdk.ValAddress(proposerAcc.GetAddress()), proposerPubKey)
	appState, _ := BinanceAppGenState(app.Codec, []json.RawMessage{genTx})
	appGenState, _ := wire.MarshalJSONIndent(app.Codec, appState)
	app.InitChain(abci.RequestInitChain{AppStateBytes: appGenState})
	app.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: 42, Time: time.Unix(0, 100), ProposerAddress: proposerValAddr}})
	app.SetCheckState(abci.Header{Height: 42, Time: time.Unix(0, 100), ProposerAddress: proposerValAddr})

	proposer := abci.Validator{Address: proposerValAddr, Power: 10}
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: proposerValAddr}).WithVoteInfos([]abci.VoteInfo{
		{Validator: proposer, SignedLastBlock: true},
	})

	app.publicationConfig = &config.PublicationConfig{
		PublishOrderUpdates:    true,
		PublishAccountBalance:  true,
		PublishOrderBook:       true,
		PublishBlockFee:        true,
		PublicationChannelSize: 0, // deliberately sync publication
	}
	app.publisher = pub.NewMockMarketDataPublisher(logger, app.publicationConfig)
	//ctx = app.NewContext(false, abci.Header{ChainID: "mychainid"})
	ctx = app.DeliverState.Ctx
	cdc = app.GetCodec()
	keeper = app.DexKeeper
	keeper.CollectOrderInfoForPublish = true
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 102000)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)
	tradingPair = dextypes.NewTradingPair("ZCB", "BNB", 102000)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)
	keeper.FeeManager.FeeConfig.ExpireFee = expireFee
	keeper.FeeManager.FeeConfig.IOCExpireFee = iocExpireFee
	keeper.FeeManager.FeeConfig.FeeRate = 1000
	keeper.FeeManager.FeeConfig.FeeRateNative = 500
	keeper.FeeManager.FeeConfig.CancelFee = 12
	keeper.FeeManager.FeeConfig.CancelFeeNative = 6

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

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	publisher.Lock.Lock()
	require.Len(publisher.BooksPublished, 1)
	require.Len(publisher.BooksPublished[0].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ_BNB", []pub.PriceLevel{{102000, 3000000}}, make([]pub.PriceLevel, 0)}, publisher.BooksPublished[0].Books[0])
	publisher.Lock.Unlock()
}

func TestAppPub_MatchOrder(t *testing.T) {
	assert, require := setupAppTest(t)

	msg := orderPkg.NewNewOrderMsg(buyer, orderPkg.GenerateOrderID(1, buyer), orderPkg.Side.BUY, "XYZ_BNB", 102000, 300000000)
	app.SetDeliverState(abci.Header{Height: 41, Time: time.Unix(0, 100)})
	handler := orderPkg.NewHandler(cdc, keeper, am)
	buyerAcc.SetSequence(1)
	am.SetAccount(ctx, buyerAcc)
	ctx = ctx.WithValue(baseapp.TxHashKey, "")
	res := handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 41})

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	publisher.Lock.Lock()
	require.Len(publisher.BooksPublished, 1)
	require.Len(publisher.AccountPublished, 1)
	require.Len(publisher.AccountPublished[0].Accounts, 1)
	expectedAccountToPub := pub.Account{string(buyer), "", []*pub.AssetBalance{{"BNB", 99999694000, 0, 306000}, {"XYZ", 100000000000, 0, 0}}}
	require.Equal(expectedAccountToPub, publisher.AccountPublished[0].Accounts[0])
	publisher.Lock.Unlock()

	// we add a sell order to fully execute the buyer order
	msg = orderPkg.NewNewOrderMsg(seller, orderPkg.GenerateOrderID(1, seller), orderPkg.Side.SELL, "XYZ_BNB", 102000, 400000000)
	app.SetDeliverState(abci.Header{Height: 42, Time: time.Unix(0, 101)})
	sellerAcc.SetSequence(1)
	am.SetAccount(ctx, sellerAcc)
	res = handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 42})

	publisher.Lock.Lock()
	require.Len(publisher.BooksPublished, 2)
	require.Len(publisher.BooksPublished[1].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ_BNB", []pub.PriceLevel{{102000, 0}}, []pub.PriceLevel{{102000, 100000000}}}, publisher.BooksPublished[1].Books[0])
	expectedAccountToPub = pub.Account{string(buyer), "BNB:153", []*pub.AssetBalance{{"BNB", 99999693847, 0, 0}, {"XYZ", 100300000000, 0, 0}}}
	expectedAccountToPubSeller := pub.Account{string(seller), "BNB:153", []*pub.AssetBalance{{"BNB", 100000305847, 0, 0}, {"XYZ", 99600000000, 0, 100000000}}}
	require.Len(publisher.AccountPublished, 2)
	require.Len(publisher.AccountPublished[1].Accounts, 3) // including the validator's account
	require.Contains(publisher.AccountPublished[1].Accounts, expectedAccountToPub)
	require.Contains(publisher.AccountPublished[1].Accounts, expectedAccountToPubSeller)
	publisher.Lock.Unlock()

	// we execute qty 1000000 sell order but add a new qty 1000000 sell order, both buy and sell price level should not publish
	msg = orderPkg.NewNewOrderMsg(buyer, orderPkg.GenerateOrderID(2, buyer), orderPkg.Side.BUY, "XYZ_BNB", 102000, 100000000)
	app.SetDeliverState(abci.Header{Height: 43, Time: time.Unix(0, 102)})
	buyerAcc.SetSequence(2)
	am.SetAccount(ctx, buyerAcc)
	res = handler(ctx, msg)
	msg = orderPkg.NewNewOrderMsg(seller, orderPkg.GenerateOrderID(2, seller), orderPkg.Side.SELL, "XYZ_BNB", 102000, 100000000)
	sellerAcc.SetSequence(2)
	am.SetAccount(ctx, sellerAcc)
	res = handler(ctx, msg)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 43})
	expectedAccountToPub = pub.Account{string(buyer), "BNB:51", []*pub.AssetBalance{{"BNB", 99999897949, 0, 0}, {"XYZ", 100100000000, 0, 0}}}
	expectedAccountToPubSeller = pub.Account{string(seller), "BNB:51", []*pub.AssetBalance{{"BNB", 100000101949, 0, 0}, {"XYZ", 99900000000, 0, 0}}}

	publisher.Lock.Lock()
	require.Len(publisher.BooksPublished, 3)
	require.Len(publisher.BooksPublished[2].Books, 0)
	require.Len(publisher.AccountPublished, 3)
	require.Len(publisher.AccountPublished[2].Accounts, 3) // including the validator's account
	require.Contains(publisher.AccountPublished[2].Accounts, expectedAccountToPub)
	require.Contains(publisher.AccountPublished[2].Accounts, expectedAccountToPubSeller)
	publisher.Lock.Unlock()
}

func TestAppPub_MatchAndCancelFee(t *testing.T) {
	assert, require := setupAppTest(t)
	handler := orderPkg.NewHandler(cdc, keeper, am)

	// ==== Place a to-be-matched sell order and a to-be-cancelled buy order (in different symbol)
	msg := orderPkg.NewNewOrderMsg(seller, orderPkg.GenerateOrderID(1, seller), orderPkg.Side.SELL, "XYZ_BNB", 102000, 100000000)
	app.SetDeliverState(abci.Header{Height: 41, Time: time.Unix(0, 100)})
	sellerAcc.SetSequence(1)
	am.SetAccount(ctx, sellerAcc)
	ctx = ctx.WithValue(baseapp.TxHashKey, "")
	res := handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)

	msg2 := orderPkg.NewNewOrderMsg(buyer, orderPkg.GenerateOrderID(1, buyer), orderPkg.Side.BUY, "ZCB_BNB", 102000, 100000000)
	buyerAcc.SetSequence(1)
	am.SetAccount(ctx, buyerAcc)
	res = handler(ctx, msg2)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)

	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 41})

	// ==== Place a must-match buy order and a cancel message
	msg3 := orderPkg.NewNewOrderMsg(buyer, orderPkg.GenerateOrderID(2, buyer), orderPkg.Side.BUY, "XYZ_BNB", 102000, 100000000)
	app.SetDeliverState(abci.Header{Height: 42, Time: time.Unix(0, 101)})
	buyerAcc = am.GetAccount(ctx, buyer)
	buyerAcc.SetSequence(2)
	am.SetAccount(ctx, buyerAcc)
	res = handler(ctx, msg3)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)

	cxlMsg := orderPkg.NewCancelOrderMsg(buyer, "ZCB_BNB", orderPkg.GenerateOrderID(1, buyer), orderPkg.GenerateOrderID(1, buyer))
	buyerAcc = am.GetAccount(ctx, buyer)
	buyerAcc.SetSequence(3)
	am.SetAccount(ctx, buyerAcc)
	res = handler(ctx, cxlMsg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)

	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 42})

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	publisher.Lock.Lock()
	require.Len(publisher.TradesAndOrdersPublished, 2)
	assert.Equal("BNB:51", publisher.TradesAndOrdersPublished[1].Trades.Trades[0].Sfee)
	assert.Equal("BNB:57;#Cxl:1", publisher.TradesAndOrdersPublished[1].Trades.Trades[0].Bfee)
	assert.Equal("BNB:108", publisher.BlockFeePublished[1].Fee)
	publisher.Lock.Unlock()
}
