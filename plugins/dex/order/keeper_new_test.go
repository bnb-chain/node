package order

import (
	"os"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	dextypes "github.com/binance-chain/node/plugins/dex/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
)

func TestNewKeeper_MatchFailure(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	accAdd, _ := MakeAddress()
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	tradingPair.LotSize = -10000000 // negative LotSize should never happen
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ-000_BNB", 99000, 3000000)
	ord := OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(ord, false)
	msg = NewNewOrderMsg(accAdd, "123457", Side.BUY, "XYZ-000_BNB", 99000, 1000000)
	ord = OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(ord, false)
	msg = NewNewOrderMsg(accAdd, "123458", Side.BUY, "XYZ-000_BNB", 99000, 5000000)
	ord = OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(ord, false)
	msg = NewNewOrderMsg(accAdd, "123459", Side.SELL, "XYZ-000_BNB", 98000, 1000000)
	ord = OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(ord, false)
	msg = NewNewOrderMsg(accAdd, "123460", Side.SELL, "XYZ-000_BNB", 97000, 5000000)
	ord = OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(ord, false)
	msg = NewNewOrderMsg(accAdd, "123461", Side.SELL, "XYZ-000_BNB", 95000, 5000000)
	ord = OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(ord, false)
	msg = NewNewOrderMsg(accAdd, "123462", Side.BUY, "XYZ-000_BNB", 99000, 15000000)
	ord = OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(ord, false)
	tradeOuts := keeper.matchAndDistributeTrades(true, 42, 0)
	c := channelHash(accAdd, 4)
	i := 0
	for tr := range tradeOuts[c] {
		assert.Equal(tr.eventType, eventCancelForMatchFailure)
		assert.Equal(tr.in, tr.out)
		assert.Equal(tr.in, tr.unlock)
		i++
	}
	assert.Equal(7, i)
}

func TestNewKeeper_MarkBreatheBlock(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	tt, _ := time.Parse(time.RFC3339, "2018-01-02T15:04:05Z")
	keeper.MarkBreatheBlock(ctx, 42, tt)
	h := keeper.GetLastBreatheBlockHeight(ctx, 42, tt, 0, 10)
	assert.Equal(int64(42), h)
	tt.AddDate(0, 0, 9)
	h = keeper.GetLastBreatheBlockHeight(ctx, 42, tt, 0, 10)
	assert.Equal(int64(42), h)
	tt, _ = time.Parse(time.RFC3339, "2018-01-03T15:04:05Z")
	keeper.MarkBreatheBlock(ctx, 43, tt)
	h = keeper.GetLastBreatheBlockHeight(ctx, 42, tt, 0, 10)
	assert.Equal(int64(43), h)
	tt.AddDate(0, 0, 9)
	h = keeper.GetLastBreatheBlockHeight(ctx, 42, tt, 0, 10)
	assert.Equal(int64(43), h)
}

func TestNew_compressAndSave(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	//keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)

	l := me.NewOrderBookOnULList(7, 3)
	l.InsertOrder("123451", me.SELLSIDE, 10000, 9950, 1000)
	l.InsertOrder("123452", me.SELLSIDE, 10000, 9955, 1000)
	l.InsertOrder("123453", me.SELLSIDE, 10001, 10000, 1000)
	l.InsertOrder("123454", me.SELLSIDE, 10002, 10000, 1000)
	l.InsertOrder("123455", me.SELLSIDE, 10002, 10010, 1000)
	l.InsertOrder("123456", me.SELLSIDE, 10002, 10020, 1000)
	l.InsertOrder("123457", me.SELLSIDE, 10003, 10020, 1000)
	l.InsertOrder("123458", me.SELLSIDE, 10003, 10021, 1000)
	l.InsertOrder("123459", me.SELLSIDE, 10004, 10022, 1000)
	l.InsertOrder("123460", me.SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123461", me.SELLSIDE, 10005, 10030, 1000)
	l.InsertOrder("123462", me.SELLSIDE, 10005, 10032, 1000)
	l.InsertOrder("123463", me.SELLSIDE, 10005, 10033, 1000)
	buys, sells := l.GetAllLevels()
	snapshot := OrderBookSnapshot{Buys: buys, Sells: sells, LastTradePrice: 100}
	bytes, _ := cdc.MarshalBinaryLengthPrefixed(snapshot)
	t.Logf("before compress, size is %v", len(bytes))
	kvstore := cms.GetKVStore(common.DexStoreKey)
	key := "testcompress"
	compressAndSave(snapshot, cdc, key, kvstore)
	bz := kvstore.Get([]byte(key))
	assert.NotNil(bz)
	t.Logf("after compress, size is %v", len(bz))
	assert.True(len(bz) < len(bytes))
}

func TestNewKeeper_SnapShotOrderBook(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	accAdd, _ := MakeAddress()
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ-000_BNB", 102000, 3000000)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(accAdd, "123457", Side.BUY, "XYZ-000_BNB", 101000, 1000000)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(accAdd, "123458", Side.BUY, "XYZ-000_BNB", 99000, 5000000)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(accAdd, "123459", Side.SELL, "XYZ-000_BNB", 980000, 1000000)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(accAdd, "123460", Side.SELL, "XYZ-000_BNB", 970000, 5000000)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(accAdd, "123461", Side.SELL, "XYZ-000_BNB", 950000, 5000000)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(accAdd, "123462", Side.BUY, "XYZ-000_BNB", 96000, 1500000)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	keeper.RoundOrderExists("XYZ-000_BNB", Side.BUY, 96000, "123462")
	assert.Equal(1, len(keeper.allOrders))
	assert.Equal(7, len(keeper.allOrders["XYZ-000_BNB"]))
	assert.Equal(1, len(keeper.engines))

	keeper.matchAndDistributeTrades(false, 42, 0)

	effectedStoredKeys1, err := keeper.SnapShotOrderBook(ctx, 43)
	t.Log(err)
	storedKVPairs1 := effectedStoredKVPairs(keeper, ctx, effectedStoredKeys1)
	effectedStoredKeys2, err := keeper.SnapShotOrderBook(ctx, 43)
	t.Log(err)
	storedKVPairs2 := effectedStoredKVPairs(keeper, ctx, effectedStoredKeys2)

	assert.Equal(storedKVPairs1, storedKVPairs2)

	assert.Nil(err)
	keeper.MarkBreatheBlock(ctx, 43, time.Now())
	keeper2 := MakeKeeper(cdc)
	h, err := keeper2.LoadOrderBookSnapshot(ctx, 43, utils.Now(), 0, 10)
	assert.Equal(7, len(keeper2.allOrders["XYZ-000_BNB"]))
	o123459 := keeper2.allOrders["XYZ-000_BNB"]["123459"]
	assert.Equal(int64(980000), o123459.Price)
	assert.Equal(int64(1000000), o123459.Quantity)
	assert.Equal(int64(0), o123459.CumQty)
	assert.Equal(int64(42), o123459.CreatedHeight)
	assert.Equal(int64(84), o123459.CreatedTimestamp)
	assert.Equal(int64(42), o123459.LastUpdatedHeight)
	assert.Equal(int64(84), o123459.LastUpdatedTimestamp)
	assert.Equal(1, len(keeper2.engines))
	assert.Equal(int64(43), h)
	buys, sells := keeper2.engines["XYZ-000_BNB"].Book.GetAllLevels()
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))
	assert.Equal(int64(102000), buys[0].Price)
	assert.Equal(int64(96000), buys[3].Price)
	assert.Equal(int64(950000), sells[0].Price)
	assert.Equal(int64(980000), sells[2].Price)
}

func TestNewKeeper_SnapShotAndLoadAfterMatch(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	accAdd, _ := MakeAddress()
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg123456 := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ-000_BNB", 102000, 3000000)
	info123456 := OrderInfo{msg123456, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(info123456, false)
	msg123457 := NewNewOrderMsg(accAdd, "123457", Side.BUY, "XYZ-000_BNB", 10000, 1000000)
	info123457 := OrderInfo{msg123457, 42, 0, 42, 0, 0, "", 0}
	keeper.AddOrder(info123457, false)
	msg := NewNewOrderMsg(accAdd, "123458", Side.SELL, "XYZ-000_BNB", 100000, 2000000)
	keeper.AddOrder(OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}, false)
	assert.Equal(1, len(keeper.allOrders))
	assert.Equal(3, len(keeper.allOrders["XYZ-000_BNB"]))
	assert.Equal(1, len(keeper.engines))

	keeper.MatchAll(42, 0)
	_, err := keeper.SnapShotOrderBook(ctx, 43)
	assert.Nil(err)
	keeper.MarkBreatheBlock(ctx, 43, time.Now())
	keeper2 := MakeKeeper(cdc)
	h, err := keeper2.LoadOrderBookSnapshot(ctx, 43, utils.Now(), 0, 10)
	assert.Equal(2, len(keeper2.allOrders["XYZ-000_BNB"]))
	assert.Equal(int64(102000), keeper2.allOrders["XYZ-000_BNB"]["123456"].Price)
	assert.Equal(int64(2000000), keeper2.allOrders["XYZ-000_BNB"]["123456"].CumQty)
	assert.Equal(int64(10000), keeper2.allOrders["XYZ-000_BNB"]["123457"].Price)
	assert.Equal(int64(0), keeper2.allOrders["XYZ-000_BNB"]["123457"].CumQty)
	info123456_Changed := info123456
	info123456_Changed.CumQty = 2000000
	assert.Equal(info123456_Changed, *keeper2.OrderInfosForPub["123456"])
	assert.Equal(info123457, *keeper2.OrderInfosForPub["123457"])
	assert.Equal(1, len(keeper2.engines))
	assert.Equal(int64(102000), keeper2.engines["XYZ-000_BNB"].LastTradePrice)
	assert.Equal(int64(43), h)
	buys, sells := keeper2.engines["XYZ-000_BNB"].Book.GetAllLevels()
	assert.Equal(2, len(buys))
	assert.Equal(0, len(sells))
	assert.Equal(int64(102000), buys[0].Price)
}

func TestNewKeeper_SnapShotOrderBookEmpty(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	accAdd, _ := MakeAddress()

	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ-000_BNB", 102000, 300000)
	keeper.AddOrder(OrderInfo{msg, 42, 0, 42, 0, 0, "", 0}, false)
	assert.True(keeper.RoundOrderExists("XYZ-000_BNB", Side.BUY, 102000, "123456"))
	keeper.RemoveOrder(msg.Id, msg.Symbol, nil)
	assert.False(keeper.RoundOrderExists("XYZ-000_BNB", Side.BUY, 102000, "123456"))
	buys, sells := keeper.engines["XYZ-000_BNB"].Book.GetAllLevels()
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	_, err := keeper.SnapShotOrderBook(ctx, 43)
	assert.Nil(err)
	keeper.MarkBreatheBlock(ctx, 43, time.Now())

	keeper2 := MakeKeeper(cdc)
	h, err := keeper2.LoadOrderBookSnapshot(ctx, 43, utils.Now(), 0, 10)
	assert.Equal(int64(43), h)
	assert.Equal(0, len(keeper2.allOrders["XYZ-000_BNB"]))
	buys, sells = keeper2.engines["XYZ-000_BNB"].Book.GetAllLevels()
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
}

func TestNewKeeper_LoadOrderBookSnapshot(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)

	keeper.PairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("XYZ-000", "BNB", 1e8))
	h, err := keeper.LoadOrderBookSnapshot(ctx, 0, utils.Now(), 0, 10)
	assert.Zero(h)
	assert.Nil(err)
}

func TestNewKeeper_ReplayOrdersFromBlock(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	memDB := db.NewMemDB()
	blockStore, stateDB := GenerateBlocksAndSave(memDB, false, cdc)
	logger := log.NewTMLogger(os.Stdout)
	cms := MakeCMS(nil)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	err := keeper.ReplayOrdersFromBlock(ctx, blockStore, stateDB, int64(3), int64(1), auth.DefaultTxDecoder(cdc))
	assert.Nil(err)
	buys, sells := keeper.engines["XYZ-000_BNB"].Book.GetAllLevels()
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(98000), sells[0].Price)
	assert.Equal(int64(97000), buys[0].Price)
	assert.Equal(int64(1000000), buys[0].Orders[0].CumQty)
	assert.Equal(int64(96000), buys[1].Price)
}

func TestNewKeeper_ReplayOrdersFromBlockWithInvalidTx(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	memDB := db.NewMemDB()
	blockStore, stateDB := GenerateBlocksAndSave(memDB, true, cdc)
	logger := log.NewTMLogger(os.Stdout)
	cms := MakeCMS(nil)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	err := keeper.ReplayOrdersFromBlock(ctx, blockStore, stateDB, int64(3), int64(1), auth.DefaultTxDecoder(cdc))
	assert.Nil(err)
	buys, sells := keeper.engines["XYZ-000_BNB"].Book.GetAllLevels()
	assert.Equal(1, len(buys))
	assert.Equal(2, len(sells))
	assert.Equal(int64(97000), sells[0].Price)
	assert.Equal(int64(96000), buys[0].Price)
	assert.Equal(int64(0), buys[0].Orders[0].CumQty)
}

func TestNewKeeper_InitOrderBookDay1(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	memDB := db.NewMemDB()
	blockStore, stateDB := GenerateBlocksAndSave(memDB, false, cdc)
	logger := log.NewTMLogger(os.Stdout)
	cms := MakeCMS(memDB)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	keeper2 := MakeKeeper(cdc)
	//blockStore := tmstore.NewBlockStore(memDB)
	ctx = sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	keeper2.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper2.initOrderBook(ctx, 0, 7, blockStore, stateDB, 3, auth.DefaultTxDecoder(cdc))
	buys, sells := keeper2.engines["XYZ-000_BNB"].Book.GetAllLevels()
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(98000), sells[0].Price)
	assert.Equal(int64(97000), buys[0].Price)
	assert.Equal(int64(1000000), buys[0].Orders[0].CumQty)
	assert.Equal(int64(96000), buys[1].Price)
}

func TestNewKeeper_ExpireOrders(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, am, keeper := setup()
	keeper.FeeManager.UpdateConfig(NewTestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 0)
	addr := acc.GetAddress()
	keeper.AddEngine(dextypes.NewTradingPair("ABC-000", "BNB", 1e6))
	keeper.AddEngine(dextypes.NewTradingPair("XYZ-000", "BNB", 1e6))
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "1", Side.BUY, "ABC-000_BNB", 1e6, 1e6), 10000, 0, 10000, 0, 0, "", 0}, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "2", Side.BUY, "ABC-000_BNB", 2e6, 2e6), 10000, 0, 10000, 0, 0, "", 0}, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "3", Side.BUY, "XYZ-000_BNB", 1e6, 2e6), 10000, 0, 10000, 0, 0, "", 0}, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "4", Side.SELL, "ABC-000_BNB", 3e6, 1e8), 10000, 0, 10000, 0, 0, "", 0}, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "5", Side.SELL, "ABC-000_BNB", 3e6, 2e8), 15000, 0, 15000, 0, 0, "", 0}, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "6", Side.BUY, "XYZ-000_BNB", 2e6, 2e6), 20000, 0, 20000, 0, 0, "", 0}, false)
	acc.(types.NamedAccount).SetLockedCoins(sdk.Coins{
		sdk.NewCoin("ABC-000", 3e8),
		sdk.NewCoin("BNB", 11e4),
	}.Sort())
	am.SetAccount(ctx, acc)

	breathTime, _ := time.Parse(time.RFC3339, "2018-01-02T00:00:01Z")
	keeper.matchAndDistributeTrades(false, 42, 0)
	keeper.MarkBreatheBlock(ctx, 15000, breathTime)

	keeper.ExpireOrders(ctx, breathTime.AddDate(0, 0, 3), nil)
	buys, sells := keeper.engines["ABC-000_BNB"].Book.GetAllLevels()
	require.Len(t, buys, 0)
	require.Len(t, sells, 1)
	require.Len(t, sells[0].Orders, 1)
	require.Equal(t, int64(2e8), sells[0].TotalLeavesQty())
	require.Len(t, keeper.allOrders["ABC-000_BNB"], 1)
	buys, sells = keeper.engines["XYZ-000_BNB"].Book.GetAllLevels()
	require.Len(t, buys, 1)
	require.Len(t, sells, 0)
	require.Len(t, buys[0].Orders, 1)
	require.Equal(t, int64(2e6), buys[0].TotalLeavesQty())
	require.Len(t, keeper.allOrders["XYZ-000_BNB"], 1)
	expectFees := types.NewFee(sdk.Coins{
		sdk.NewCoin("BNB", 6e4),
		sdk.NewCoin("ABC-000", 1e7),
	}.Sort(), types.FeeForProposer)
	require.Equal(t, expectFees, fees.Pool.BlockFees())
	acc = am.GetAccount(ctx, acc.GetAddress())
	require.Equal(t, sdk.Coins{
		sdk.NewCoin("ABC-000", 2e8),
		sdk.NewCoin("BNB", 4e4),
	}.Sort(), acc.(types.NamedAccount).GetLockedCoins())
	require.Equal(t, sdk.Coins{
		sdk.NewCoin("ABC-000", 9e7),
		sdk.NewCoin("BNB", 1e4),
	}.Sort(), acc.GetCoins())
	fees.Pool.Clear()
}

func TestNewKeeper_DetermineLotSize(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	ctx, _, keeper := setup()
	lotsize := keeper.DetermineLotSize("BNB", "BTC-000", 1e6)
	assert.Equal(int64(1e5), lotsize)
	lotsize = keeper.DetermineLotSize("AAA-000", "BNB", 1e6)
	assert.Equal(int64(1e7), lotsize)

	// no recentPrices recorded, use engine.LastTradePrice
	pair1 := dextypes.NewTradingPairWithLotSize("BNB", "BTC-000", 1e6, 1e5)
	keeper.AddEngine(pair1)
	pair2 := dextypes.NewTradingPairWithLotSize("AAA-000", "BNB", 1e6, 1e7)
	keeper.AddEngine(pair2)
	lotsize = keeper.DetermineLotSize("AAA-000", "BTC-000", 1e4)
	assert.Equal(int64(1e7), lotsize)
	lotsize = keeper.DetermineLotSize("BTC-000", "AAA-000", 1e12)
	assert.Equal(int64(1e3), lotsize)

	// store some recentPrices
	keeper.StoreTradePrices(ctx.WithBlockHeight(1 * pricesStoreEvery))
	keeper.engines[pair1.GetSymbol()].LastTradePrice = 1e8
	keeper.engines[pair2.GetSymbol()].LastTradePrice = 1e8
	keeper.StoreTradePrices(ctx.WithBlockHeight(2 * pricesStoreEvery))
	lotsize = keeper.DetermineLotSize("AAA-000", "BTC-000", 1e4)
	assert.Equal(int64(1e6), lotsize) // wma price of AAA-000/BNB is between 1e7 and 1e8
	lotsize = keeper.DetermineLotSize("BTC-000", "AAA-000", 1e12)
	assert.Equal(int64(1e5), lotsize) // wma price of BNB/BTC-000 is between 1e7 and 1e8
}

func TestNewKeeper_UpdateTickSizeAndLotSize(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	ctx, _, keeper := setup()
	upgrade.Mgr.AddUpgradeHeight(upgrade.LotSizeOptimization, -1)

	pair1 := dextypes.NewTradingPairWithLotSize("BNB", "BTC-000", 1e5, 1e5)
	keeper.AddEngine(pair1)
	assert.NoError(keeper.PairMapper.AddTradingPair(ctx, pair1))
	pair2 := dextypes.NewTradingPairWithLotSize("AAA-000", "BNB", 1e5, 1e8)
	keeper.AddEngine(pair2)
	assert.NoError(keeper.PairMapper.AddTradingPair(ctx, pair2))
	pair3 := dextypes.NewTradingPairWithLotSize("AAA-000", "BTC-000", 1e2, 1e8)
	keeper.AddEngine(pair3)
	assert.NoError(keeper.PairMapper.AddTradingPair(ctx, pair3))

	for i := 0; i < minimalNumPrices; i++ {
		keeper.engines[pair1.GetSymbol()].LastTradePrice += 1e5
		keeper.engines[pair2.GetSymbol()].LastTradePrice += 1e5
		keeper.engines[pair3.GetSymbol()].LastTradePrice += 1e2
		keeper.StoreTradePrices(ctx.WithBlockHeight(int64(i) * pricesStoreEvery))
	}
	keeper.UpdateTickSizeAndLotSize(ctx)
	assert.Equal(int64(1e5), keeper.engines[pair1.GetSymbol()].LotSize)
	assert.Equal(int64(1e6), keeper.engines[pair2.GetSymbol()].LotSize)
	assert.Equal(int64(1e6), keeper.engines[pair3.GetSymbol()].LotSize)
	pair1, err := keeper.PairMapper.GetTradingPair(ctx, pair1.BaseAssetSymbol, pair1.QuoteAssetSymbol)
	assert.NoError(err)
	assert.Equal(int64(1e2), pair1.TickSize.ToInt64())
	assert.Equal(int64(1e5), pair1.LotSize.ToInt64())
	pair2, err = keeper.PairMapper.GetTradingPair(ctx, pair2.BaseAssetSymbol, pair2.QuoteAssetSymbol)
	assert.NoError(err)
	assert.Equal(int64(1e2), pair2.TickSize.ToInt64())
	assert.Equal(int64(1e6), pair2.LotSize.ToInt64())
	pair3, err = keeper.PairMapper.GetTradingPair(ctx, pair3.BaseAssetSymbol, pair3.QuoteAssetSymbol)
	assert.NoError(err)
	assert.Equal(int64(1), pair3.TickSize.ToInt64())
	assert.Equal(int64(1e6), pair3.LotSize.ToInt64())
}

func TestNewKeeper_UpdateLotSize(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	logger := log.NewTMLogger(os.Stdout)
	cms := MakeCMS(nil)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeCheck, logger)
	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	keeper.UpdateLotSize(tradingPair.GetSymbol(), 1e3)

	assert.Equal(int64(1e3), keeper.engines[tradingPair.GetSymbol()].LotSize)
}

func TestNewOpenOrders_AfterMatch(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	keeper := initKeeper()
	keeper.AddEngine(dextypes.NewTradingPair("NNB", "BNB", 100000000))

	// add an original buy order, waiting to be filled
	msg := NewNewOrderMsg(zc, ZcAddr+"-0", Side.BUY, "NNB_BNB", 1000000000, 1000000000)
	orderInfo := OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}
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
	orderInfo = OrderInfo{msg, 43, 86, 43, 86, 0, "", 0}
	keeper.AddOrder(orderInfo, false)
	res = keeper.GetOpenOrders("NNB_BNB", zz)
	assert.Equal(1, len(res))

	// match existing two orders
	keeper.MatchAll(43, 86)

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
	orderInfo = OrderInfo{msg, 44, 88, 44, 88, 0, "", 0}
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
	keeper.MatchAll(44, 88)

	// after match, all orders should be closed
	res = keeper.GetOpenOrders("NNB_BNB", zc)
	assert.Equal(0, len(res))
	res = keeper.GetOpenOrders("NNB_BNB", zz)
	assert.Equal(0, len(res))
}

func TestNewKeeper_DelistTradingPair(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	ctx, am, keeper := setup()
	fees.Pool.Clear()
	keeper.FeeManager.UpdateConfig(NewTestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 0)
	addr := acc.GetAddress()

	tradingPair := dextypes.NewTradingPair("XYZ-000", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	acc.(types.NamedAccount).SetLockedCoins(sdk.Coins{
		sdk.NewCoin("BNB", 11e4),
		sdk.NewCoin("XYZ-000", 4e4),
	}.Sort())

	acc.(types.NamedAccount).SetCoins(sdk.Coins{
		sdk.NewCoin("XYZ-000", 4e5),
	}.Sort())

	am.SetAccount(ctx, acc)

	msg := NewNewOrderMsg(addr, "123456", Side.BUY, "XYZ-000_BNB", 1e6, 1e6)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "1234562", Side.BUY, "XYZ-000_BNB", 1e6, 1e6)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "123457", Side.BUY, "XYZ-000_BNB", 2e6, 1e6)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "123458", Side.BUY, "XYZ-000_BNB", 3e6, 1e6)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "123459", Side.SELL, "XYZ-000_BNB", 5e6, 1e4)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "123460", Side.SELL, "XYZ-000_BNB", 6e6, 1e4)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "1234602", Side.SELL, "XYZ-000_BNB", 6e6, 1e4)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "123461", Side.SELL, "XYZ-000_BNB", 7e6, 1e4)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	msg = NewNewOrderMsg(addr, "123462", Side.BUY, "XYZ-000_BNB", 4e6, 1e6)
	keeper.AddOrder(OrderInfo{msg, 42, 84, 42, 84, 0, "", 0}, false)
	assert.Equal(1, len(keeper.allOrders))
	assert.Equal(9, len(keeper.allOrders["XYZ-000_BNB"]))
	assert.Equal(1, len(keeper.engines))

	keeper.matchAndDistributeTrades(false, 42, 0)
	assert.Equal(1, len(keeper.allOrders))
	assert.Equal(9, len(keeper.allOrders["XYZ-000_BNB"]))
	assert.Equal(1, len(keeper.engines))
	keeper.DelistTradingPair(ctx, "XYZ-000_BNB", nil)
	assert.Equal(0, len(keeper.allOrders))
	assert.Equal(0, len(keeper.engines))

	expectFees := types.NewFee(sdk.Coins{
		sdk.NewCoin("BNB", 10e4),
		sdk.NewCoin("XYZ-000", 4e5),
	}.Sort(), types.FeeForProposer)
	require.Equal(t, expectFees, fees.Pool.BlockFees())
}

//
func TestNewKeeper_DelistTradingPair_Empty(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	assert := assert.New(t)
	ctx, _, keeper := setup()
	fees.Pool.Clear()
	keeper.FeeManager.UpdateConfig(NewTestFeeConfig())

	tradingPair := dextypes.NewTradingPair("XYZ-001", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	assert.Equal(1, len(keeper.allOrders))
	assert.Equal(0, len(keeper.allOrders["XYZ-001_BNB"]))
	assert.Equal(1, len(keeper.engines))

	keeper.DelistTradingPair(ctx, "XYZ-001_BNB", nil)
	assert.Equal(0, len(keeper.allOrders))
	assert.Equal(0, len(keeper.engines))

	expectFees := types.NewFee(sdk.Coins(nil), types.ZeroFee)
	require.Equal(t, expectFees, fees.Pool.BlockFees())
}

func TestNewKeeper_CanListTradingPair_Normal(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, _, keeper := setup()

	err := keeper.CanListTradingPair(ctx, "AAA-000", types.NativeTokenSymbol)
	require.Nil(t, err)

	err = keeper.CanListTradingPair(ctx, types.NativeTokenSymbol, "AAA-000")
	require.Nil(t, err)
}

func TestNewKeeper_CanListTradingPair_Abnormal(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, _, keeper := setup()

	err := keeper.CanListTradingPair(ctx, "AAA-000", "AAA-000")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "base asset symbol should not be identical to quote asset symbol")

	err = keeper.CanListTradingPair(ctx, "BBB-000", "AAA-000")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "token BBB-000 should be listed against BNB before against AAA-000")

	err = keeper.PairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("BBB-000", types.NativeTokenSymbol, 1e8))
	require.Nil(t, err)

	err = keeper.CanListTradingPair(ctx, "BBB-000", "AAA-000")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "token AAA-000 should be listed against BNB before listing BBB-000 against AAA-000")
}

func TestNewKeeper_CanDelistTradingPair(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, _, keeper := setup()

	err := keeper.CanDelistTradingPair(ctx, "AAA-000", "AAA-000")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "base asset symbol should not be identical to quote asset symbol")

	err = keeper.PairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("BBB-000", types.NativeTokenSymbol, 1e8))
	err = keeper.CanDelistTradingPair(ctx, "BBB-000", types.NativeTokenSymbol)
	require.Nil(t, err)

	err = keeper.PairMapper.AddTradingPair(ctx, dextypes.NewTradingPair(types.NativeTokenSymbol, "BBB-000", 1e8))
	err = keeper.CanDelistTradingPair(ctx, types.NativeTokenSymbol, "BBB-000")
	require.Nil(t, err)

	err = keeper.PairMapper.AddTradingPair(ctx, dextypes.NewTradingPair(types.NativeTokenSymbol, "BBB-000", 1e8))
	err = keeper.PairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("BBB-000", "AAA-000", 1e8))
	require.Nil(t, err)

	err = keeper.CanDelistTradingPair(ctx, types.NativeTokenSymbol, "BBB-000")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "trading pair BBB-000_AAA-000 should not exist before delisting BNB_BBB-000")
}

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP19, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}
