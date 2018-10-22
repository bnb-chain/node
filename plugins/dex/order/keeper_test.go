package order

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	sdkstore "github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	abci "github.com/tendermint/tendermint/abci/types"
	bc "github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	me "github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	wire.RegisterCrypto(cdc) // Register crypto.
	bank.RegisterWire(cdc)
	sdk.RegisterWire(cdc) // Register Msgs
	tokens.RegisterWire(cdc)
	types.RegisterWire(cdc)
	cdc.RegisterConcrete(NewOrderMsg{}, "dex/NewOrder", nil)
	cdc.RegisterConcrete(CancelOrderMsg{}, "dex/CancelOrder", nil)

	cdc.RegisterConcrete(OrderBookSnapshot{}, "dex/OrderBookSnapshot", nil)
	cdc.RegisterConcrete(ActiveOrders{}, "dex/ActiveOrders", nil)

	return cdc
}

func MakeKeeper(cdc *wire.Codec) *Keeper {
	accountMapper := auth.NewAccountMapper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	coinKeeper := bank.NewKeeper(accountMapper)
	codespacer := sdk.NewCodespacer()
	pairMapper := store.NewTradingPairMapper(cdc, common.PairStoreKey)
	keeper := NewKeeper(common.DexStoreKey, coinKeeper, pairMapper,
		codespacer.RegisterNext(dextypes.DefaultCodespace), 2, cdc, false)
	return keeper
}

func MakeCMS(memDB *db.MemDB) sdk.CacheMultiStore {
	if memDB == nil {
		memDB = db.NewMemDB()
	}
	ms := sdkstore.NewCommitMultiStore(memDB)
	ms.MountStoreWithDB(common.DexStoreKey, sdk.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(common.PairStoreKey, sdk.StoreTypeIAVL, nil)
	ms.LoadLatestVersion()
	cms := ms.CacheMultiStore()
	return cms
}

func TestKeeper_MatchFailure(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	accAdd, _ := MakeAddress()
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	tradingPair.LotSize = -10000000 // negative LotSize should never happen
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ_BNB", 99000, 3000000)
	ord := OrderInfo{msg, 100000, 0, msg.Id}
	keeper.AddOrder(ord, 42, false)
	msg = NewNewOrderMsg(accAdd, "123457", Side.BUY, "XYZ_BNB", 99000, 1000000)
	ord = OrderInfo{msg, 100001, 0, msg.Id}
	keeper.AddOrder(ord, 42, false)
	msg = NewNewOrderMsg(accAdd, "123458", Side.BUY, "XYZ_BNB", 99000, 5000000)
	ord = OrderInfo{msg, 100002, 0, msg.Id}
	keeper.AddOrder(ord, 42, false)
	msg = NewNewOrderMsg(accAdd, "123459", Side.SELL, "XYZ_BNB", 98000, 1000000)
	ord = OrderInfo{msg, 100003, 0, msg.Id}
	keeper.AddOrder(ord, 42, false)
	msg = NewNewOrderMsg(accAdd, "123460", Side.SELL, "XYZ_BNB", 97000, 5000000)
	ord = OrderInfo{msg, 100004, 0, msg.Id}
	keeper.AddOrder(ord, 42, false)
	msg = NewNewOrderMsg(accAdd, "123461", Side.SELL, "XYZ_BNB", 95000, 5000000)
	ord = OrderInfo{msg, 100005, 0, msg.Id}
	keeper.AddOrder(ord, 42, false)
	msg = NewNewOrderMsg(accAdd, "123462", Side.BUY, "XYZ_BNB", 99000, 15000000)
	ord = OrderInfo{msg, 100006, 0, msg.Id}
	keeper.AddOrder(ord, 42, false)
	tradeOuts := keeper.matchAndDistributeTrades(true)
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

func TestKeeper_MarkBreatheBlock(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	tt, _ := time.Parse(time.RFC3339, "2018-01-02T15:04:05Z")
	ts := tt.Unix()
	keeper.MarkBreatheBlock(ctx, 42, ts)
	h := keeper.getLastBreatheBlockHeight(ctx, tt, 10)
	assert.Equal(int64(42), h)
	tt.AddDate(0, 0, 9)
	h = keeper.getLastBreatheBlockHeight(ctx, tt, 10)
	assert.Equal(int64(42), h)
	tt, _ = time.Parse(time.RFC3339, "2018-01-03T15:04:05Z")
	ts = tt.Unix()
	keeper.MarkBreatheBlock(ctx, 43, ts)
	h = keeper.getLastBreatheBlockHeight(ctx, tt, 10)
	assert.Equal(int64(43), h)
	tt.AddDate(0, 0, 9)
	h = keeper.getLastBreatheBlockHeight(ctx, tt, 10)
	assert.Equal(int64(43), h)
}

func Test_compressAndSave(t *testing.T) {
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
	bytes, _ := cdc.MarshalBinary(snapshot)
	t.Logf("before compress, size is %v", len(bytes))
	kvstore := cms.GetKVStore(common.DexStoreKey)
	key := "testcompress"
	compressAndSave(snapshot, cdc, key, kvstore)
	bz := kvstore.Get([]byte(key))
	assert.NotNil(bz)
	t.Logf("after compress, size is %v", len(bz))
	assert.True(len(bz) < len(bytes))
}

func MakeAddress() (sdk.AccAddress, secp256k1.PrivKeySecp256k1) {
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	return addr, privKey
}

func effectedStoredKVPairs(keeper *Keeper, ctx sdk.Context, keys []string) map[string][]byte {
	res := make(map[string][]byte, len(keys))
	store := ctx.KVStore(keeper.storeKey)
	for _, key := range keys {
		res[key] = store.Get([]byte(key))
	}
	return res
}

func TestKeeper_SnapShotOrderBook(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	accAdd, _ := MakeAddress()
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	msg = NewNewOrderMsg(accAdd, "123457", Side.BUY, "XYZ_BNB", 101000, 1000000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	msg = NewNewOrderMsg(accAdd, "123458", Side.BUY, "XYZ_BNB", 99000, 5000000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	msg = NewNewOrderMsg(accAdd, "123459", Side.SELL, "XYZ_BNB", 98000, 1000000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	msg = NewNewOrderMsg(accAdd, "123460", Side.SELL, "XYZ_BNB", 97000, 5000000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	msg = NewNewOrderMsg(accAdd, "123461", Side.SELL, "XYZ_BNB", 95000, 5000000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	msg = NewNewOrderMsg(accAdd, "123462", Side.BUY, "XYZ_BNB", 96000, 1500000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	assert.Equal(1, len(keeper.allOrders))
	assert.Equal(7, len(keeper.allOrders["XYZ_BNB"]))
	assert.Equal(1, len(keeper.engines))

	effectedStoredKeys1, err := keeper.SnapShotOrderBook(ctx, 43)
	storedKVPairs1 := effectedStoredKVPairs(keeper, ctx, effectedStoredKeys1)
	effectedStoredKeys2, err := keeper.SnapShotOrderBook(ctx, 43)
	storedKVPairs2 := effectedStoredKVPairs(keeper, ctx, effectedStoredKeys2)
	assert.Equal(storedKVPairs1, storedKVPairs2)

	assert.Nil(err)
	keeper.MarkBreatheBlock(ctx, 43, time.Now().Unix())
	keeper2 := MakeKeeper(cdc)
	h, err := keeper2.LoadOrderBookSnapshot(ctx, 10)
	assert.Equal(7, len(keeper2.allOrders["XYZ_BNB"]))
	assert.Equal(int64(98000), keeper2.allOrders["XYZ_BNB"]["123459"].Price)
	assert.Equal(1, len(keeper2.engines))
	assert.Equal(int64(43), h)
	buys, sells := keeper2.engines["XYZ_BNB"].Book.GetAllLevels()
	assert.Equal(4, len(buys))
	assert.Equal(3, len(sells))
	assert.Equal(int64(102000), buys[0].Price)
	assert.Equal(int64(96000), buys[3].Price)
	assert.Equal(int64(95000), sells[0].Price)
	assert.Equal(int64(98000), sells[2].Price)
}

func TestKeeper_SnapShotOrderBookEmpty(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	accAdd, _ := MakeAddress()

	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ_BNB", 102000, 300000)
	keeper.AddOrder(OrderInfo{msg, 0, 0, ""}, 42, false)
	keeper.CancelOrder(msg.Id, msg.Symbol, nil)
	buys, sells := keeper.engines["XYZ_BNB"].Book.GetAllLevels()
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	_, err := keeper.SnapShotOrderBook(ctx, 43)
	assert.Nil(err)
	keeper.MarkBreatheBlock(ctx, 43, time.Now().Unix())

	keeper2 := MakeKeeper(cdc)
	h, err := keeper2.LoadOrderBookSnapshot(ctx, 10)
	assert.Equal(int64(43), h)
	assert.Equal(0, len(keeper2.allOrders["XYZ_BNB"]))
	buys, sells = keeper2.engines["XYZ_BNB"].Book.GetAllLevels()
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
}

func TestKeeper_LoadOrderBookSnapshot(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)

	keeper.PairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("XYZ", "BNB", 1e8))
	h, err := keeper.LoadOrderBookSnapshot(ctx, 10)
	assert.Zero(h)
	assert.Nil(err)
}

func NewMockBlock(txs []auth.StdTx, height int64, commit *tmtypes.Commit, cdc *wire.Codec) *tmtypes.Block {
	tmTxs := make([]tmtypes.Tx, len(txs))
	for i, tx := range txs {
		tmTxs[i], _ = cdc.MarshalBinary(tx)
	}
	return tmtypes.MakeBlock(height, tmTxs, commit, nil)
}

const BlockPartSize = 65536

func MakeTxFromMsg(msgs []sdk.Msg, accountNumber, seqNum int64, privKey secp256k1.PrivKeySecp256k1) auth.StdTx {
	fee, _ := sdk.ParseCoin("100 BNB")
	signMsg := auth.StdSignMsg{
		ChainID:       "chainID1",
		AccountNumber: accountNumber,
		Sequence:      seqNum,
		Msgs:          msgs,
		Memo:          "Memo1",
		Fee:           auth.NewStdFee(int64(100), fee), // TODO run simulate to estimate gas?
	}
	sig, _ := privKey.Sign(signMsg.Bytes())
	sigs := []auth.StdSignature{{
		PubKey:        privKey.PubKey(),
		Signature:     sig,
		AccountNumber: accountNumber,
		Sequence:      seqNum,
	}}
	tx := auth.NewStdTx(signMsg.Msgs, signMsg.Fee, sigs, signMsg.Memo)
	return tx
}

func GenerateBlocksAndSave(storedb db.DB, cdc *wire.Codec) *bc.BlockStore {
	blockStore := bc.NewBlockStore(storedb)
	lastCommit := &tmtypes.Commit{}
	buyerAdd, buyerPrivKey := MakeAddress()
	sellerAdd, sellerPrivKey := MakeAddress()
	txs := make([]auth.StdTx, 0)
	height := int64(1)
	block := NewMockBlock(txs, height, lastCommit, cdc)
	blockParts := block.MakePartSet(BlockPartSize)
	blockStore.SaveBlock(block, blockParts, &tmtypes.Commit{})
	height++
	txs = make([]auth.StdTx, 7)
	msgs01 := []sdk.Msg{NewNewOrderMsg(buyerAdd, "123456", Side.BUY, "XYZ_BNB", 102000, 3000000)}
	txs[0] = MakeTxFromMsg(msgs01, int64(100), int64(9001), buyerPrivKey)
	msgs02 := []sdk.Msg{NewNewOrderMsg(buyerAdd, "123457", Side.BUY, "XYZ_BNB", 101000, 1000000)}
	txs[1] = MakeTxFromMsg(msgs02, int64(100), int64(9002), buyerPrivKey)
	msgs03 := []sdk.Msg{NewNewOrderMsg(sellerAdd, "123459", Side.SELL, "XYZ_BNB", 98000, 1000000)}
	txs[2] = MakeTxFromMsg(msgs03, int64(1001), int64(7001), sellerPrivKey)
	msgs04 := []sdk.Msg{NewNewOrderMsg(buyerAdd, "123458", Side.BUY, "XYZ_BNB", 99000, 5000000)}
	txs[3] = MakeTxFromMsg(msgs04, int64(100), int64(9003), buyerPrivKey)
	msgs05 := []sdk.Msg{NewNewOrderMsg(sellerAdd, "123460", Side.SELL, "XYZ_BNB", 97000, 5000000)}
	txs[4] = MakeTxFromMsg(msgs05, int64(1001), int64(7002), sellerPrivKey)
	msgs06 := []sdk.Msg{NewNewOrderMsg(sellerAdd, "123461", Side.SELL, "XYZ_BNB", 95000, 5000000)}
	txs[5] = MakeTxFromMsg(msgs06, int64(1001), int64(7003), sellerPrivKey)
	msgs07 := []sdk.Msg{NewNewOrderMsg(buyerAdd, "123462", Side.BUY, "XYZ_BNB", 96000, 1500000)}
	txs[6] = MakeTxFromMsg(msgs07, int64(100), int64(9004), buyerPrivKey)
	block = NewMockBlock(txs, height, lastCommit, cdc)
	blockParts = block.MakePartSet(BlockPartSize)
	blockStore.SaveBlock(block, blockParts, &tmtypes.Commit{})
	//blockID := tmtypes.BlockID{Hash: block.Hash(), PartsHeader: blockParts.Header()}
	//lastCommit = tmtypes.MakeCommit(block)
	height++
	msgs11 := []sdk.Msg{NewNewOrderMsg(buyerAdd, "123463", Side.BUY, "XYZ_BNB", 96000, 2500000)}
	msgs12 := []sdk.Msg{NewNewOrderMsg(buyerAdd, "123464", Side.BUY, "XYZ_BNB", 97000, 1500000)}
	msgs13 := []sdk.Msg{NewNewOrderMsg(sellerAdd, "123465", Side.SELL, "XYZ_BNB", 107000, 1500000)}
	msgs14 := []sdk.Msg{NewCancelOrderMsg(buyerAdd, "XYZ_BNB", "123466", "123462")}
	msgs15 := []sdk.Msg{NewCancelOrderMsg(sellerAdd, "XYZ_BNB", "123467", "123465")}
	txs = make([]auth.StdTx, 5)
	txs[0] = MakeTxFromMsg(msgs11, int64(100), int64(9005), buyerPrivKey)
	txs[1] = MakeTxFromMsg(msgs12, int64(100), int64(9006), buyerPrivKey)
	txs[2] = MakeTxFromMsg(msgs13, int64(100), int64(7004), sellerPrivKey)
	txs[3] = MakeTxFromMsg(msgs14, int64(100), int64(9007), buyerPrivKey)
	txs[4] = MakeTxFromMsg(msgs15, int64(100), int64(7005), sellerPrivKey)
	block = NewMockBlock(txs, height, lastCommit, cdc)
	blockParts = block.MakePartSet(BlockPartSize)
	blockStore.SaveBlock(block, blockParts, &tmtypes.Commit{})
	return blockStore
}

func TestKeeper_ReplayOrdersFromBlock(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	memDB := db.NewMemDB()
	blockStore := GenerateBlocksAndSave(memDB, cdc)
	logger := log.NewTMLogger(os.Stdout)
	cms := MakeCMS(nil)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	err := keeper.ReplayOrdersFromBlock(blockStore, int64(3), int64(1), tx.DefaultTxDecoder(cdc))
	assert.Nil(err)
	buys, sells := keeper.engines["XYZ_BNB"].Book.GetAllLevels()
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(98000), sells[0].Price)
	assert.Equal(int64(97000), buys[0].Price)
	assert.Equal(int64(1000000), buys[0].Orders[0].CumQty)
	assert.Equal(int64(96000), buys[1].Price)
}

func TestKeeper_InitOrderBookDay1(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	memDB := db.NewMemDB()
	GenerateBlocksAndSave(memDB, cdc)
	logger := log.NewTMLogger(os.Stdout)
	cms := MakeCMS(memDB)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	keeper2 := MakeKeeper(cdc)
	ctx = sdk.NewContext(cms, abci.Header{}, true, logger)
	keeper2.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper2.InitOrderBook(ctx, 7, memDB, 3, tx.DefaultTxDecoder(cdc))
	buys, sells := keeper2.engines["XYZ_BNB"].Book.GetAllLevels()
	assert.Equal(2, len(buys))
	assert.Equal(1, len(sells))
	assert.Equal(int64(98000), sells[0].Price)
	assert.Equal(int64(97000), buys[0].Price)
	assert.Equal(int64(1000000), buys[0].Orders[0].CumQty)
	assert.Equal(int64(96000), buys[1].Price)
}

func setup() (ctx sdk.Context, mapper auth.AccountMapper, keeper *Keeper) {
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	types.RegisterWire(cdc)
	wire.RegisterCrypto(cdc)
	mapper = auth.NewAccountMapper(cdc, capKey, types.ProtoAppAccount)
	ctx = sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())
	coinKeeper := bank.NewKeeper(mapper)
	keeper = NewKeeper(capKey2, coinKeeper, nil, sdk.NewCodespacer().RegisterNext(dextypes.DefaultCodespace), 2, cdc, false)
	return
}

func TestKeeper_CalcOrderFees(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.FeeManager.UpdateConfig(ctx, TestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 0)

	// InCcy == BNB
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeToken, 0)})
	tran := Transfer{
		eventType:  eventFilled,
		accAddress: acc.GetAddress(),
		inAsset:    types.NativeToken,
		in:         100e8,
		outAsset:   "ABC",
		out:        1000e8,
		unlock:     1000e8,
	}
	fee := keeper.calcOrderFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 5e6)}, fee.Tokens)

	// InCcy != BNB
	keeper.engines["ABC_"+types.NativeToken] = me.NewMatchEng(1e7, 1, 1)
	_, acc = testutils.NewAccount(ctx, am, 100)
	tran = Transfer{
		eventType:  eventFilled,
		accAddress: acc.GetAddress(),
		inAsset:    "ABC",
		in:         1000e8,
		outAsset:   types.NativeToken,
		out:        100e8,
		unlock:     110e8,
	}
	// has enough bnb
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeToken, 1e8)})
	fee = keeper.calcOrderFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 5e6)}, fee.Tokens)
	// no enough bnb
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeToken, 1e6)})
	fee = keeper.calcOrderFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC", 1e8)}, fee.Tokens)
}

func TestKeeper_CalcExpireFee(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.FeeManager.UpdateConfig(ctx, TestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 0)
	// in BNB
	tran := Transfer{
		eventType:  eventFullyExpire,
		accAddress: acc.GetAddress(),
		inAsset:    types.NativeToken,
		in:         100e8,
		outAsset:   types.NativeToken,
		out:        100e8,
		unlock:     100e8,
	}
	fee := keeper.calcFixedFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 10000)}, fee.Tokens)
	tran.eventType = eventIOCFullyExpire
	fee = keeper.calcFixedFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 5000)}, fee.Tokens)
	tran.out, tran.in, tran.out = 4000, 4000, 4000
	fee = keeper.calcFixedFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 4000)}, fee.Tokens)

	// sell ABC, and BNB as quote asset
	tran = Transfer{
		eventType:  eventFullyExpire,
		accAddress: acc.GetAddress(),
		inAsset:    "ABC",
		in:         1000e8,
		outAsset:   "ABC",
		out:        1000e8,
		unlock:     1000e8,
	}
	keeper.engines["ABC_"+types.NativeToken] = me.NewMatchEng(1e7, 1, 1)
	fee = keeper.calcFixedFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC", 100000)}, fee.Tokens)
	tran = Transfer{
		eventType:  eventFullyExpire,
		accAddress: acc.GetAddress(),
		inAsset:    "ABC",
		in:         900,
		outAsset:   "ABC",
		out:        900,
		unlock:     900,
	}
	fee = keeper.calcFixedFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC", 900)}, fee.Tokens)

	// sell BTC, BNB as base asset
	tran = Transfer{
		eventType:  eventFullyExpire,
		accAddress: acc.GetAddress(),
		inAsset:    "BTC",
		in:         1e5,
		outAsset:   "BTC",
		out:        1e5,
		unlock:     1e5,
	}
	keeper.engines[types.NativeToken+"_BTC"] = me.NewMatchEng(1e5, 1, 1)
	fee = keeper.calcFixedFee(ctx, acc, tran)
	require.Equal(t, sdk.Coins{sdk.NewCoin("BTC", 10)}, fee.Tokens)
}

func TestKeeper_ExpireOrders(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.FeeManager.UpdateConfig(ctx, TestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 0)
	addr := acc.GetAddress()
	keeper.AddEngine(dextypes.NewTradingPair("ABC", "BNB", 1e6))
	keeper.AddEngine(dextypes.NewTradingPair("XYZ", "BNB", 1e6))
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "1", Side.BUY, "ABC_BNB", 1e6, 1e8), 0, 0, ""}, 10000, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "2", Side.BUY, "ABC_BNB", 2e6, 2e8), 0, 0, ""}, 10000, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "3", Side.BUY, "XYZ_BNB", 1e6, 1e8), 0, 0, ""}, 10000, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "4", Side.SELL, "ABC_BNB", 1e6, 1e8), 0, 0, ""}, 10000, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "5", Side.SELL, "ABC_BNB", 2e6, 2e8), 0, 0, ""}, 15000, false)
	keeper.AddOrder(OrderInfo{NewNewOrderMsg(addr, "6", Side.BUY, "XYZ_BNB", 2e6, 2e8), 0, 0, ""}, 20000, false)
	acc.(types.NamedAccount).SetLockedCoins(sdk.Coins{
		sdk.NewCoin("ABC", 3e8),
		sdk.NewCoin("BNB", 10e6),
	}.Sort())
	am.SetAccount(ctx, acc)

	breathTime, _ := time.Parse(time.RFC3339, "2018-01-02T00:00:01Z")
	keeper.MarkBreatheBlock(ctx, 15000, breathTime.Unix())

	ctx, _, err := keeper.ExpireOrders(ctx, breathTime.AddDate(0, 0, 3).Unix(), am, nil)
	require.NoError(t, err)
	buys, sells := keeper.engines["ABC_BNB"].Book.GetAllLevels()
	require.Len(t, buys, 0)
	require.Len(t, sells, 1)
	require.Len(t, sells[0].Orders, 1)
	require.Equal(t, int64(2e8), sells[0].TotalLeavesQty())
	require.Len(t, keeper.allOrders["ABC_BNB"], 1)
	buys, sells = keeper.engines["XYZ_BNB"].Book.GetAllLevels()
	require.Len(t, buys, 1)
	require.Len(t, sells, 0)
	require.Len(t, buys[0].Orders, 1)
	require.Equal(t, int64(2e8), buys[0].TotalLeavesQty())
	require.Len(t, keeper.allOrders["XYZ_BNB"], 1)
	expectFees := types.NewFee(sdk.Coins{
		sdk.NewCoin("BNB", 30000),
		sdk.NewCoin("ABC", 1000000),
	}.Sort(), types.FeeForProposer)
	require.Equal(t, expectFees, tx.Fee(ctx))
	acc = am.GetAccount(ctx, acc.GetAddress())
	require.Equal(t, sdk.Coins{
		sdk.NewCoin("ABC", 2e8),
		sdk.NewCoin("BNB", 4e6),
	}.Sort(), acc.(types.NamedAccount).GetLockedCoins())
	require.Equal(t, sdk.Coins{
		sdk.NewCoin("ABC", 99e6),
		sdk.NewCoin("BNB", 597e4),
	}.Sort(), acc.GetCoins())
}

func TestKeeper_UpdateLotSize(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	logger := log.NewTMLogger(os.Stdout)
	cms := MakeCMS(nil)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	keeper.UpdateLotSize(tradingPair.GetSymbol(), 1e3)

	assert.Equal(int64(1e3), keeper.engines[tradingPair.GetSymbol()].LotSize)
}
