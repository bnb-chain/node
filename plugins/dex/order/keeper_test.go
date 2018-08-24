package order

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	sdkstore "github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	abci "github.com/tendermint/tendermint/abci/types"
	bc "github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/BiJie/BinanceChain/common"
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
	keeper, _ := NewKeeper(common.DexStoreKey, coinKeeper, pairMapper,
		codespacer.RegisterNext(dextypes.DefaultCodespace), 2, cdc)
	return keeper
}

func MakeCMS() sdk.CacheMultiStore {
	memDB := db.NewMemDB()
	ms := sdkstore.NewCommitMultiStore(memDB)
	ms.MountStoreWithDB(common.DexStoreKey, sdk.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(common.PairStoreKey, sdk.StoreTypeIAVL, nil)
	ms.LoadLatestVersion()
	cms := ms.CacheMultiStore()
	return cms
}
func TestKeeper_MarkBreatheBlock(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS()
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	tt, _ := time.Parse(time.RFC3339, "2018-01-02T15:04:05Z")
	ts := tt.Unix()
	keeper.MarkBreatheBlock(ctx, 42, ts)
	kvstore := ctx.KVStore(common.DexStoreKey)
	h := keeper.GetBreatheBlockHeight(tt, kvstore, 10)
	assert.Equal(int64(42), h)
	tt.AddDate(0, 0, 9)
	h = keeper.GetBreatheBlockHeight(tt, kvstore, 10)
	assert.Equal(int64(42), h)
	tt, _ = time.Parse(time.RFC3339, "2018-01-03T15:04:05Z")
	ts = tt.Unix()
	keeper.MarkBreatheBlock(ctx, 43, ts)
	h = keeper.GetBreatheBlockHeight(tt, kvstore, 10)
	assert.Equal(int64(43), h)
	tt.AddDate(0, 0, 9)
	h = keeper.GetBreatheBlockHeight(tt, kvstore, 10)
	assert.Equal(int64(43), h)
}

func Test_compressAndSave(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	//keeper := MakeKeeper(cdc)
	cms := MakeCMS()

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

func MakeAddress() (sdk.AccAddress, ed25519.PrivKeyEd25519) {
	privKey := ed25519.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	return addr, privKey
}

func TestKeeper_SnapShotOrderBook(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS()
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	accAdd, _ := MakeAddress()
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(msg, 42)
	msg = NewNewOrderMsg(accAdd, "123457", Side.BUY, "XYZ_BNB", 101000, 1000000)
	keeper.AddOrder(msg, 42)
	msg = NewNewOrderMsg(accAdd, "123458", Side.BUY, "XYZ_BNB", 99000, 5000000)
	keeper.AddOrder(msg, 42)
	msg = NewNewOrderMsg(accAdd, "123459", Side.SELL, "XYZ_BNB", 98000, 1000000)
	keeper.AddOrder(msg, 42)
	msg = NewNewOrderMsg(accAdd, "123460", Side.SELL, "XYZ_BNB", 97000, 5000000)
	keeper.AddOrder(msg, 42)
	msg = NewNewOrderMsg(accAdd, "123461", Side.SELL, "XYZ_BNB", 95000, 5000000)
	keeper.AddOrder(msg, 42)
	msg = NewNewOrderMsg(accAdd, "123462", Side.BUY, "XYZ_BNB", 96000, 1500000)
	keeper.AddOrder(msg, 42)
	assert.Equal(7, len(keeper.allOrders))
	assert.Equal(1, len(keeper.engines))
	err := keeper.SnapShotOrderBook(ctx, 43)
	assert.Nil(err)
	keeper.MarkBreatheBlock(ctx, 43, time.Now().Unix())
	keeper2 := MakeKeeper(cdc)
	h, err := keeper2.LoadOrderBookSnapshot(ctx, 10)
	assert.Equal(7, len(keeper2.allOrders))
	assert.Equal(int64(98000), keeper2.allOrders["123459"].Price)
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
	cms := MakeCMS()
	logger := log.NewTMLogger(os.Stdout)
	ctx := sdk.NewContext(cms, abci.Header{}, true, logger)
	accAdd, _ := MakeAddress()

	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	msg := NewNewOrderMsg(accAdd, "123456", Side.BUY, "XYZ_BNB", 102000, 300000)
	keeper.AddOrder(msg, 42)
	keeper.RemoveOrder(msg.Id, msg.Symbol, msg.Side, msg.Price)
	buys, sells := keeper.engines["XYZ_BNB"].Book.GetAllLevels()
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
	err := keeper.SnapShotOrderBook(ctx, 43)
	assert.Nil(err)
	keeper.MarkBreatheBlock(ctx, 43, time.Now().Unix())

	keeper2 := MakeKeeper(cdc)
	h, err := keeper2.LoadOrderBookSnapshot(ctx, 10)
	assert.Equal(int64(43), h)
	assert.Equal(0, len(keeper2.allOrders))
	buys, sells = keeper2.engines["XYZ_BNB"].Book.GetAllLevels()
	assert.Equal(0, len(buys))
	assert.Equal(0, len(sells))
}

func TestKeeper_LoadOrderBookSnapshot(t *testing.T) {
	assert := assert.New(t)
	cdc := MakeCodec()
	keeper := MakeKeeper(cdc)
	cms := MakeCMS()
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

func MakeTxFromMsg(msgs []sdk.Msg, accountNumber, seqNum int64, privKey ed25519.PrivKeyEd25519) auth.StdTx {
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
	msgs14 := []sdk.Msg{NewCancelOrderMsg(buyerAdd, "123466", "123462")}
	msgs15 := []sdk.Msg{NewCancelOrderMsg(sellerAdd, "123467", "123465")}
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
	cms := MakeCMS()
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
