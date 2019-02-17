package app

import (
	"encoding/json"
	"os"
	"sync/atomic"
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

	"github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/app/pub"
	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/testutils"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
	dextypes "github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/wire"
)

const (
	expireFee    = 1000
	iocExpireFee = 500
)

func prepareGenTx(cdc *codec.Codec, chainId string,
	valOperAddr sdk.ValAddress, valPubKey crypto.PubKey) json.RawMessage {
	msg := stake.MsgCreateValidatorProposal{
		MsgCreateValidator: stake.NewMsgCreateValidator(
			valOperAddr,
			valPubKey,
			DefaultSelfDelegationToken,
			stake.NewDescription("pub", "", "", ""),
			stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
		),
	}
	tx := auth.NewStdTx([]sdk.Msg{msg}, nil, "", 0, nil)
	txBytes, err := wire.MarshalJSONIndent(cdc, tx)
	if err != nil {
		panic(err)
	}

	return txBytes
}

func setupAppTest(t *testing.T) (*assert.Assertions, *require.Assertions, *BinanceChain, sdk.Account, sdk.Account) {
	logger := log.NewNopLogger()
	db := dbm.NewMemDB()

	app := NewBinanceChain(logger, db, os.Stdout)
	app.SetAnteHandler(nil)
	app.SetDeliverState(abci.Header{})
	am := app.AccountKeeper
	ctx := app.NewContext(sdk.RunTxModeDeliver, abci.Header{})

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
	app.DeliverState.Ctx = app.DeliverState.Ctx.WithBlockHeader(abci.Header{ProposerAddress: proposerValAddr}).WithVoteInfos([]abci.VoteInfo{
		{Validator: proposer, SignedLastBlock: true},
	})
	am.SetAccount(app.DeliverState.Ctx, proposerAcc)

	app.publicationConfig = &config.PublicationConfig{
		PublishOrderUpdates:    true,
		PublishAccountBalance:  true,
		PublishOrderBook:       true,
		PublishBlockFee:        true,
		PublicationChannelSize: 0, // deliberately sync publication
	}
	pub.Logger = logger.With("module", "pub")
	pub.Cfg = app.publicationConfig
	pub.ToPublishCh = make(chan pub.BlockInfoToPublish, app.publicationConfig.PublicationChannelSize)
	app.publisher = pub.NewMockMarketDataPublisher()
	go pub.Publish(app.publisher, app.metrics, logger, app.publicationConfig, pub.ToPublishCh)
	pub.IsLive = true

	keeper := app.DexKeeper
	keeper.CollectOrderInfoForPublish = true
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 102000)
	keeper.PairMapper.AddTradingPair(app.DeliverState.Ctx, tradingPair)
	keeper.AddEngine(tradingPair)
	tradingPair = dextypes.NewTradingPair("ZCB-000", "BNB", 102000)
	keeper.PairMapper.AddTradingPair(app.DeliverState.Ctx, tradingPair)
	keeper.AddEngine(tradingPair)
	keeper.FeeManager.FeeConfig.ExpireFee = expireFee
	keeper.FeeManager.FeeConfig.IOCExpireFee = iocExpireFee
	keeper.FeeManager.FeeConfig.FeeRate = 1000
	keeper.FeeManager.FeeConfig.FeeRateNative = 500
	keeper.FeeManager.FeeConfig.CancelFee = 12
	keeper.FeeManager.FeeConfig.CancelFeeNative = 6

	_, buyerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 0, 0) // give user enough coins to pay the fee
	_, sellerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 0, 0)
	return assert.New(t), require.New(t), app, buyerAcc, sellerAcc
}

func TestAppPub_AddOrder(t *testing.T) {
	assert, require, app, buyerAcc, _ := setupAppTest(t)

	msg := orderPkg.NewNewOrderMsg(buyerAcc.GetAddress(), "1", orderPkg.Side.BUY, "XYZ-000_BNB", 102000, 3000000)
	app.DexKeeper.AddOrder(orderPkg.OrderInfo{msg, 42, 0, 42, 0, 0, ""}, false)
	app.EndBlocker(app.DeliverState.Ctx, abci.RequestEndBlock{Height: 42})

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	for 4 != atomic.LoadUint32(&publisher.MessagePublished) {
		time.Sleep(1000)
	}
	publisher.Lock.Lock()
	require.Len(publisher.BooksPublished, 1)
	require.Len(publisher.BooksPublished[0].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ-000_BNB", []pub.PriceLevel{{102000, 3000000}}, make([]pub.PriceLevel, 0)}, publisher.BooksPublished[0].Books[0])
	publisher.Lock.Unlock()
}

func TestAppPub_MatchOrder(t *testing.T) {
	assert, require, app, buyerAcc, sellerAcc := setupAppTest(t)

	ctx := app.DeliverState.Ctx
	msg := orderPkg.NewNewOrderMsg(buyerAcc.GetAddress(), orderPkg.GenerateOrderID(1, buyerAcc.GetAddress()), orderPkg.Side.BUY, "XYZ-000_BNB", 102000, 300000000)
	handler := orderPkg.NewHandler(app.GetCodec(), app.DexKeeper, app.AccountKeeper)
	app.DeliverState.Ctx = app.DeliverState.Ctx.WithBlockHeight(41).WithBlockTime(time.Unix(0, 100))
	buyerAcc.SetSequence(1)
	app.AccountKeeper.SetAccount(ctx, buyerAcc)
	ctx = ctx.WithValue(baseapp.TxHashKey, "")
	res := handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 41})

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	for 4 != atomic.LoadUint32(&publisher.MessagePublished) {
		time.Sleep(1000)
	}
	publisher.Lock.Lock()
	require.Len(publisher.BooksPublished, 1)
	require.Len(publisher.AccountPublished, 1)
	require.Len(publisher.AccountPublished[0].Accounts, 1)
	expectedAccountToPub := pub.Account{string(buyerAcc.GetAddress()), "", []*pub.AssetBalance{{"BNB", 99999694000, 0, 306000}, {"XYZ-000", 100000000000, 0, 0}}}
	require.Equal(expectedAccountToPub, publisher.AccountPublished[0].Accounts[0])
	publisher.Lock.Unlock()

	// we add a sell order to fully execute the buyer order
	msg = orderPkg.NewNewOrderMsg(sellerAcc.GetAddress(), orderPkg.GenerateOrderID(1, sellerAcc.GetAddress()), orderPkg.Side.SELL, "XYZ-000_BNB", 102000, 400000000)
	ctx = ctx.WithBlockHeight(42).WithBlockTime(time.Unix(0, 101))
	sellerAcc.SetSequence(1)
	app.AccountKeeper.SetAccount(ctx, sellerAcc)
	res = handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 42})
	for 8 != atomic.LoadUint32(&publisher.MessagePublished) {
		time.Sleep(1000)
	}

	publisher.Lock.Lock()
	require.Len(publisher.BooksPublished, 2)
	require.Len(publisher.BooksPublished[1].Books, 1)
	assert.Equal(pub.OrderBookDelta{"XYZ-000_BNB", []pub.PriceLevel{{102000, 0}}, []pub.PriceLevel{{102000, 100000000}}}, publisher.BooksPublished[1].Books[0])
	expectedAccountToPub = pub.Account{string(buyerAcc.GetAddress()), "BNB:153", []*pub.AssetBalance{{"BNB", 99999693847, 0, 0}, {"XYZ-000", 100300000000, 0, 0}}}
	expectedAccountToPubSeller := pub.Account{string(sellerAcc.GetAddress()), "BNB:153", []*pub.AssetBalance{{"BNB", 100000305847, 0, 0}, {"XYZ-000", 99600000000, 0, 100000000}}}
	require.Len(publisher.AccountPublished, 2)
	require.Len(publisher.AccountPublished[1].Accounts, 3) // including the validator's account
	require.Contains(publisher.AccountPublished[1].Accounts, expectedAccountToPub)
	require.Contains(publisher.AccountPublished[1].Accounts, expectedAccountToPubSeller)
	publisher.Lock.Unlock()

	// we execute qty 1000000 sell order but add a new qty 1000000 sell order, both buy and sell price level should not publish
	msg = orderPkg.NewNewOrderMsg(buyerAcc.GetAddress(), orderPkg.GenerateOrderID(2, buyerAcc.GetAddress()), orderPkg.Side.BUY, "XYZ-000_BNB", 102000, 100000000)
	ctx = ctx.WithBlockHeight(43).WithBlockTime(time.Unix(0, 102))
	buyerAcc.SetSequence(2)
	app.AccountKeeper.SetAccount(ctx, buyerAcc)
	res = handler(ctx, msg)
	msg = orderPkg.NewNewOrderMsg(sellerAcc.GetAddress(), orderPkg.GenerateOrderID(2, sellerAcc.GetAddress()), orderPkg.Side.SELL, "XYZ-000_BNB", 102000, 100000000)
	sellerAcc.SetSequence(2)
	app.AccountKeeper.SetAccount(ctx, sellerAcc)
	res = handler(ctx, msg)
	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 43})
	for 12 != atomic.LoadUint32(&publisher.MessagePublished) {
		time.Sleep(1000)
	}
	expectedAccountToPub = pub.Account{string(buyerAcc.GetAddress()), "BNB:51", []*pub.AssetBalance{{"BNB", 99999897949, 0, 0}, {"XYZ-000", 100100000000, 0, 0}}}
	expectedAccountToPubSeller = pub.Account{string(sellerAcc.GetAddress()), "BNB:51", []*pub.AssetBalance{{"BNB", 100000101949, 0, 0}, {"XYZ-000", 99900000000, 0, 0}}}

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
	assert, require, app, buyerAcc, sellerAcc := setupAppTest(t)
	handler := orderPkg.NewHandler(app.GetCodec(), app.DexKeeper, app.AccountKeeper)
	ctx := app.DeliverState.Ctx

	// ==== Place a to-be-matched sell order and a to-be-cancelled buy order (in different symbol)
	msg := orderPkg.NewNewOrderMsg(sellerAcc.GetAddress(), orderPkg.GenerateOrderID(1, sellerAcc.GetAddress()), orderPkg.Side.SELL, "XYZ-000_BNB", 102000, 100000000)
	ctx = ctx.WithBlockHeight(41).WithBlockTime(time.Unix(0, 100))
	sellerAcc.SetSequence(1)
	app.AccountKeeper.SetAccount(ctx, sellerAcc)
	ctx = ctx.WithValue(baseapp.TxHashKey, "").WithRunTxMode(sdk.RunTxModeDeliver)
	res := handler(ctx, msg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)

	msg2 := orderPkg.NewNewOrderMsg(buyerAcc.GetAddress(), orderPkg.GenerateOrderID(1, buyerAcc.GetAddress()), orderPkg.Side.BUY, "ZCB-000_BNB", 102000, 100000000)
	buyerAcc.SetSequence(1)
	app.AccountKeeper.SetAccount(ctx, buyerAcc)
	res = handler(ctx, msg2)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)

	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 41})

	// ==== Place a must-match buy order and a cancel message
	msg3 := orderPkg.NewNewOrderMsg(buyerAcc.GetAddress(), orderPkg.GenerateOrderID(2, buyerAcc.GetAddress()), orderPkg.Side.BUY, "XYZ-000_BNB", 102000, 100000000)
	ctx = ctx.WithBlockHeight(42).WithBlockTime(time.Unix(0, 101))
	buyerAcc = app.AccountKeeper.GetAccount(ctx, buyerAcc.GetAddress())
	buyerAcc.SetSequence(2)
	app.AccountKeeper.SetAccount(ctx, buyerAcc)
	res = handler(ctx, msg3)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)

	cxlMsg := orderPkg.NewCancelOrderMsg(buyerAcc.GetAddress(), "ZCB-000_BNB", orderPkg.GenerateOrderID(1, buyerAcc.GetAddress()))
	buyerAcc = app.AccountKeeper.GetAccount(ctx, buyerAcc.GetAddress())
	buyerAcc.SetSequence(3)
	app.AccountKeeper.SetAccount(ctx, buyerAcc)
	ctx = ctx.WithValue(baseapp.TxHashKey, "CANCEL1")
	res = handler(ctx, cxlMsg)
	require.Equal(sdk.ABCICodeOK, res.Code, res.Log)
	fees.Pool.CommitFee("CANCEL1")

	app.EndBlocker(ctx, abci.RequestEndBlock{Height: 42})

	publisher := app.publisher.(*pub.MockMarketDataPublisher)
	for 8 != atomic.LoadUint32(&publisher.MessagePublished) {
		time.Sleep(1000)
	}
	publisher.Lock.Lock()
	require.Len(publisher.ExecutionResultsPublished, 2)
	assert.Equal("BNB:51", publisher.ExecutionResultsPublished[1].Trades.Trades[0].Sfee)
	assert.Equal("BNB:57;#Cxl:1", publisher.ExecutionResultsPublished[1].Trades.Trades[0].Bfee)
	assert.Equal("BNB:108", publisher.BlockFeePublished[1].Fee)
	publisher.Lock.Unlock()
}
