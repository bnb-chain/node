package pub

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	pubtest "github.com/binance-chain/node/app/pub/testutils"
	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/types"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
	dextypes "github.com/binance-chain/node/plugins/dex/types"
)

// This test makes sure non-execution order changes (non partial fill and fully fill) are correctly generated

func newTestFeeConfig() orderPkg.FeeConfig {
	feeConfig := orderPkg.NewFeeConfig()
	feeConfig.FeeRateNative = 500
	feeConfig.FeeRate = 1000
	feeConfig.ExpireFeeNative = 2e4
	feeConfig.ExpireFee = 1e5
	feeConfig.IOCExpireFeeNative = 1e4
	feeConfig.IOCExpireFee = 5e4
	feeConfig.CancelFeeNative = 2e4
	feeConfig.CancelFee = 1e5
	return feeConfig
}

var keeper *orderPkg.Keeper
var buyer sdk.AccAddress
var seller sdk.AccAddress
var am auth.AccountKeeper
var ctx sdk.Context

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore, accountKey *sdk.KVStoreKey) sdk.AccountCache {
	accountStore := ms.GetKVStore(accountKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

func setupKeeperTest(t *testing.T) (*assert.Assertions, *require.Assertions) {
	cdc := pubtest.MakeCodec()
	logger := log.NewTMLogger(os.Stdout)

	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	am = auth.NewAccountKeeper(cdc, capKey, types.ProtoAppAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	ctx = sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)

	pairMapper := store.NewTradingPairMapper(cdc, common.PairStoreKey)
	keeper = orderPkg.NewKeeper(capKey2, am, pairMapper, sdk.NewCodespacer().RegisterNext(dextypes.DefaultCodespace), 2, cdc, true)
	tradingPair := dextypes.NewTradingPair("XYZ-000", types.NativeTokenSymbol, 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)
	keeper.FeeManager.UpdateConfig(newTestFeeConfig())

	keeper.FeeConfig.SetExpireFee(ctx, expireFee)
	keeper.FeeConfig.SetIOCExpireFee(ctx, iocExpireFee)
	keeper.FeeConfig.SetFeeRate(ctx, 1000)
	keeper.FeeConfig.SetFeeRateNative(ctx, 500)

	_, buyerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000) // give user enough coins to pay the fee
	buyer = buyerAcc.GetAddress()

	_, sellerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000)
	seller = sellerAcc.GetAddress()

	return assert.New(t), require.New(t)
}

func TestKeeper_AddOrder(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "1", orderPkg.Side.BUY, "XYZ-000_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "08E19B16880CF70D59DDD996E3D75C66CD0405DE", 0}, false)
	msg = orderPkg.NewNewOrderMsg(buyer, "2", orderPkg.Side.BUY, "XYZ-000_BNB", 101000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 43, 105, 43, 105, 0, "0D42245EB2BF574A5B9D485404E0E61B1A2397A9", 0}, false)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderInfosForPub, 2)
	// verify order0 - and the order in orderchanges slice
	orderChange0 := keeper.OrderChanges[0]
	assert.Equal("1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)

	// verify order1 - make sure the fields are correct
	orderInfo1 := keeper.OrderInfosForPub["2"]
	assert.Equal(buyer, orderInfo1.Sender)
	assert.Equal("2", orderInfo1.Id)
	assert.Equal("XYZ-000_BNB", orderInfo1.Symbol)
	assert.Equal(orderPkg.OrderType.LIMIT, orderInfo1.OrderType)
	assert.Equal(orderPkg.Side.BUY, orderInfo1.Side)
	assert.Equal(int64(101000), orderInfo1.Price)
	assert.Equal(int64(1000000), orderInfo1.Quantity)
	assert.Equal(orderPkg.TimeInForce.GTE, orderInfo1.TimeInForce)
	assert.Equal(int64(105), orderInfo1.CreatedTimestamp)
	assert.Equal(int64(0), orderInfo1.CumQty)
	assert.Equal("0D42245EB2BF574A5B9D485404E0E61B1A2397A9", orderInfo1.TxHash)
}

func TestKeeper_IOCExpireWithFee(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{buyer, "1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 102000, 3000000, orderPkg.TimeInForce.IOC}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "08E19B16880CF70D59DDD996E3D75C66CD0405DE", 0}, false)

	require.Len(keeper.OrderChanges, 1)
	require.Len(keeper.OrderInfosForPub, 1)

	trades := MatchAndAllocateAllForPublish(keeper, ctx)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderInfosForPub, 1)
	require.Len(trades, 0)

	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	// verify orderChange1 - IOCNofill
	assert.Equal("1", orderChange1.Id)
	assert.Equal(orderPkg.IocNoFill, orderChange1.Tpe)
	assert.Equal("BNB:10000", keeper.RoundOrderFees[string(buyer.Bytes())].String())
}

func TestKeeper_ExpireWithFee(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{buyer, "1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 102000, 3000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "08E19B16880CF70D59DDD996E3D75C66CD0405DE", 0}, false)

	require.Len(keeper.OrderChanges, 1)
	require.Len(keeper.OrderInfosForPub, 1)

	breathTime := prepareExpire(int64(43))
	ExpireOrdersForPublish(keeper, am, ctx, breathTime)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderInfosForPub, 1)

	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	// verify orderChange1 - ExpireNoFill
	assert.Equal("1", orderChange1.Id)
	assert.Equal(orderPkg.Expired, orderChange1.Tpe)
}

func TestKeeper_DelistWithFee(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{buyer, "1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 102000, 3000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "08E19B16880CF70D59DDD996E3D75C66CD0405DE", 0}, false)

	require.Len(keeper.OrderChanges, 1)
	require.Len(keeper.OrderInfosForPub, 1)

	DelistTradingPairForPublish(ctx, keeper, "XYZ-000_BNB")

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderInfosForPub, 1)

	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]

	// verify orderChange0 - Ack
	assert.Equal("1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	// verify orderChange1 - ExpireNoFill
	assert.Equal("1", orderChange1.Id)
	assert.Equal(orderPkg.Expired, orderChange1.Tpe)
}

func Test_IOCPartialExpire(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{buyer, "b-1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 100000000, 300000000, orderPkg.TimeInForce.IOC}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "", 0}, false)
	msg2 := orderPkg.NewOrderMsg{seller, "s-1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 100000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg2, 42, 100, 42, 100, 0, "", 0}, false)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderInfosForPub, 2)
	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("b-1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	// verify orderChange1 - Ack
	assert.Equal("s-1", orderChange1.Id)
	assert.Equal(orderPkg.Ack, orderChange1.Tpe)

	trades := MatchAndAllocateAllForPublish(keeper, ctx)

	require.Len(keeper.OrderChanges, 3)
	require.Len(keeper.OrderInfosForPub, 2)
	require.Len(trades, 1)
	trade0 := trades[0]
	assert.Equal("0-0", trade0.Id)
	assert.Equal("XYZ-000_BNB", trade0.Symbol)
	assert.Equal(int64(100000000), trade0.Price)
	assert.Equal(int64(100000000), trade0.Qty)
	assert.Equal("s-1", trade0.Sid)
	assert.Equal("b-1", trade0.Bid)

	orderChange2 := keeper.OrderChanges[2]
	assert.Equal("b-1", orderChange2.Id)
	assert.Equal(orderPkg.IocExpire, orderChange2.Tpe)

	assert.Equal("BNB:50000", keeper.RoundOrderFees[string(buyer.Bytes())].String())
	assert.Equal("BNB:50000", keeper.RoundOrderFees[string(seller.Bytes())].String())
}

func Test_GTEPartialExpire(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{buyer, "b-1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 100000000, 100000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "", 0}, false)
	msg2 := orderPkg.NewOrderMsg{seller, "s-1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 300000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg2, 42, 100, 42, 100, 0, "", 0}, false)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderInfosForPub, 2)
	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("b-1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	// verify orderChange1 - Ack
	assert.Equal("s-1", orderChange1.Id)
	assert.Equal(orderPkg.Ack, orderChange1.Tpe)

	trades := MatchAndAllocateAllForPublish(keeper, ctx)
	require.Len(trades, 1)
	trade0 := trades[0]
	assert.Equal("0-0", trade0.Id)
	assert.Equal("XYZ-000_BNB", trade0.Symbol)
	assert.Equal(int64(100000000), trade0.Price)
	assert.Equal(int64(100000000), trade0.Qty)
	assert.Equal("s-1", trade0.Sid)
	assert.Equal("b-1", trade0.Bid)

	assert.Equal("BNB:50000", keeper.RoundOrderFees[string(buyer.Bytes())].String())
	assert.Equal("BNB:50000", keeper.RoundOrderFees[string(seller.Bytes())].String())

	require.Len(keeper.OrderChanges, 2) // for GTE order, fully fill is not derived from transfer (will be generated by trade)
	require.Len(keeper.OrderInfosForPub, 2)

	// let the sell order expire
	breathTime := prepareExpire(int64(43))
	ExpireOrdersForPublish(keeper, am, ctx, breathTime)

	require.Len(keeper.OrderChanges, 3)
	orderChange2 := keeper.OrderChanges[2]
	assert.Equal("s-1", orderChange2.Id)
	assert.Equal(orderPkg.Expired, orderChange2.Tpe)
}

func Test_OneBuyVsTwoSell(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{buyer, "b-1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 100000000, 300000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "", 0}, false)
	msg2 := orderPkg.NewOrderMsg{seller, "s-1", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 100000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg2, 42, 100, 42, 100, 0, "", 0}, false)
	msg3 := orderPkg.NewOrderMsg{seller, "s-2", "XYZ-000_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 200000000, orderPkg.TimeInForce.GTE}
	keeper.AddOrder(orderPkg.OrderInfo{msg3, 42, 100, 42, 100, 0, "", 0}, false)

	require.Len(keeper.OrderChanges, 3)
	require.Len(keeper.OrderInfosForPub, 3)
	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	orderChange2 := keeper.OrderChanges[2]
	// verify orderChange0 - Ack
	assert.Equal("b-1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	// verify orderChange1 - Ack
	assert.Equal("s-1", orderChange1.Id)
	assert.Equal(orderPkg.Ack, orderChange1.Tpe)
	// verify orderChange2 - Ack
	assert.Equal("s-2", orderChange2.Id)
	assert.Equal(orderPkg.Ack, orderChange2.Tpe)

	trades := MatchAndAllocateAllForPublish(keeper, ctx)
	require.Len(trades, 2)
	trade0 := trades[0]
	assert.Equal("0-0", trade0.Id)
	assert.Equal("XYZ-000_BNB", trade0.Symbol)
	assert.Equal(int64(100000000), trade0.Price)
	assert.Equal(int64(100000000), trade0.Qty)
	assert.Equal("s-1", trade0.Sid)
	assert.Equal("b-1", trade0.Bid)
	trade1 := trades[1]
	assert.Equal("0-1", trade1.Id)
	assert.Equal("XYZ-000_BNB", trade1.Symbol)
	assert.Equal(int64(100000000), trade1.Price)
	assert.Equal(int64(200000000), trade1.Qty)
	assert.Equal("s-2", trade1.Sid)
	assert.Equal("b-1", trade1.Bid)

	assert.Equal("BNB:150000", keeper.RoundOrderFees[string(buyer.Bytes())].String())
	assert.Equal("BNB:150000", keeper.RoundOrderFees[string(seller.Bytes())].String())
}

func prepareExpire(height int64) time.Time {
	breathTime, _ := time.Parse(time.RFC3339, "2018-01-02T00:00:01Z")
	keeper.MarkBreatheBlock(ctx, height, breathTime)
	return breathTime.AddDate(0, 0, 3)
}
