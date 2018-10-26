package pub

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/app/config"
	pubtest "github.com/BiJie/BinanceChain/app/pub/testutils"
	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/types"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
)

// This test makes sure non-execution order changes (non partial fill and fully fill) are correctly generated

const (
	expireFee    = 1000
	iocExpireFee = 500
)

var keeper *orderPkg.Keeper
var buyer sdk.AccAddress
var seller sdk.AccAddress
var am auth.AccountMapper
var ctx sdk.Context

func setupKeeperTest(t *testing.T) (*assert.Assertions, *require.Assertions) {
	cdc := pubtest.MakeCodec()
	logger := log.NewTMLogger(os.Stdout)

	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	ctx = sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, logger)
	am = auth.NewAccountMapper(cdc, capKey, types.ProtoAppAccount)
	coinKeeper := bank.NewKeeper(am)

	pairMapper := store.NewTradingPairMapper(cdc, common.PairStoreKey)
	keeper = orderPkg.NewKeeper(capKey2, coinKeeper, pairMapper, sdk.NewCodespacer().RegisterNext(dextypes.DefaultCodespace), 2, cdc, true)
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	keeper.FeeConfig.SetExpireFee(ctx, expireFee)
	keeper.FeeConfig.SetIOCExpireFee(ctx, iocExpireFee)
	keeper.FeeConfig.SetFeeRate(ctx, 1000)
	keeper.FeeConfig.SetFeeRateNative(ctx, 500)

	_, buyerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000) // give user enough coins to pay the fee
	buyer = buyerAcc.GetAddress()

	_, sellerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000)
	seller = sellerAcc.GetAddress()

	// to get pub Logger initialized
	NewKafkaMarketDataPublisher(&config.PublicationConfig{})

	return assert.New(t), require.New(t)
}

func TestKeeper_AddOrder(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewNewOrderMsg(buyer, "1", orderPkg.Side.BUY, "XYZ_BNB", 102000, 3000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "08E19B16880CF70D59DDD996E3D75C66CD0405DE"}, false)
	msg = orderPkg.NewNewOrderMsg(buyer, "2", orderPkg.Side.BUY, "XYZ_BNB", 101000, 1000000)
	keeper.AddOrder(orderPkg.OrderInfo{msg, 43, 105, 43, 105, 0, "0D42245EB2BF574A5B9D485404E0E61B1A2397A9"}, false)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderChangesMap, 2)
	// verify order0 - and the order in orderchanges slice
	orderChange0 := keeper.OrderChanges[0]
	assert.Equal("1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	assert.Equal("", orderChange0.FeeAsset)
	assert.Equal(int64(0), orderChange0.Fee)

	// verify order1 - make sure the fields are correct
	orderInfo1 := keeper.OrderChangesMap["2"]
	assert.Equal(buyer, orderInfo1.Sender)
	assert.Equal("2", orderInfo1.Id)
	assert.Equal("XYZ_BNB", orderInfo1.Symbol)
	assert.Equal(orderPkg.OrderType.LIMIT, orderInfo1.OrderType)
	assert.Equal(orderPkg.Side.BUY, orderInfo1.Side)
	assert.Equal(int64(101000), orderInfo1.Price)
	assert.Equal(int64(1000000), orderInfo1.Quantity)
	assert.Equal(orderPkg.TimeInForce.GTC, orderInfo1.TimeInForce)
	assert.Equal(int64(105), orderInfo1.CreatedTimestamp)
	assert.Equal(int64(0), orderInfo1.CumQty)
	assert.Equal("0D42245EB2BF574A5B9D485404E0E61B1A2397A9", orderInfo1.TxHash)
}

func TestKeeper_IOCExpireWithFee(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{0x01, buyer, "1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 102000, 3000000, orderPkg.TimeInForce.IOC}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "08E19B16880CF70D59DDD996E3D75C66CD0405DE"}, false)

	require.Len(keeper.OrderChanges, 1)
	require.Len(keeper.OrderChangesMap, 1)

	trades := MatchAndAllocateAllForPublish(keeper, am, ctx)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderChangesMap, 1)
	require.Len(trades, 0)

	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	assert.Equal("", orderChange0.FeeAsset)
	assert.Equal(int64(0), orderChange0.Fee)
	// verify orderChange1 - IOCNofill
	assert.Equal("1", orderChange1.Id)
	assert.Equal(orderPkg.IocNoFill, orderChange1.Tpe)
	assert.Equal(types.NativeToken, orderChange1.FeeAsset)
	assert.Equal(int64(iocExpireFee), orderChange1.Fee)
}

func TestKeeper_ExpireWithFee(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{0x01, buyer, "1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 102000, 3000000, orderPkg.TimeInForce.GTC}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, "08E19B16880CF70D59DDD996E3D75C66CD0405DE"}, false)

	require.Len(keeper.OrderChanges, 1)
	require.Len(keeper.OrderChangesMap, 1)

	breathTime := prepareExpire(int64(43))
	ExpireOrdersForPublish(keeper, am, ctx, breathTime)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderChangesMap, 1)

	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	assert.Equal("", orderChange0.FeeAsset)
	assert.Equal(int64(0), orderChange0.Fee)
	// verify orderChange1 - ExpireNoFill
	assert.Equal("1", orderChange1.Id)
	assert.Equal(orderPkg.Expired, orderChange1.Tpe)
	assert.Equal(types.NativeToken, orderChange1.FeeAsset)
	assert.Equal(int64(expireFee), orderChange1.Fee)
}

func Test_IOCPartialExpire(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{0x01, buyer, "b-1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 100000000, 300000000, orderPkg.TimeInForce.IOC}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, ""}, false)
	msg2 := orderPkg.NewOrderMsg{0x01, seller, "s-1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 100000000, orderPkg.TimeInForce.GTC}
	keeper.AddOrder(orderPkg.OrderInfo{msg2, 42, 100, 42, 100, 0, ""}, false)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderChangesMap, 2)
	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("b-1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	assert.Equal("", orderChange0.FeeAsset)
	assert.Equal(int64(0), orderChange0.Fee)
	// verify orderChange1 - Ack
	assert.Equal("s-1", orderChange1.Id)
	assert.Equal(orderPkg.Ack, orderChange1.Tpe)
	assert.Equal("", orderChange1.FeeAsset)
	assert.Equal(int64(0), orderChange1.Fee)

	trades := MatchAndAllocateAllForPublish(keeper, am, ctx)

	require.Len(keeper.OrderChanges, 3)
	require.Len(keeper.OrderChangesMap, 2)
	require.Len(trades, 1)
	trade0 := trades[0]
	assert.Equal("0-0", trade0.Id)
	assert.Equal("XYZ_BNB", trade0.Symbol)
	assert.Equal(int64(100000000), trade0.Price)
	assert.Equal(int64(100000000), trade0.Qty)
	assert.Equal("s-1", trade0.Sid)
	assert.Equal("b-1", trade0.Bid)
	assert.Equal(int64(50000), trade0.Bfee)
	assert.Equal("BNB", trade0.BfeeAsset)
	assert.Equal(int64(50000), trade0.Sfee)
	assert.Equal("BNB", trade0.SfeeAsset)

	orderChange2 := keeper.OrderChanges[2]
	assert.Equal("b-1", orderChange2.Id)
	assert.Equal(orderPkg.IocNoFill, orderChange2.Tpe)
	assert.Equal("", orderChange2.FeeAsset)
	assert.Equal(int64(0), orderChange2.Fee) // partially filled expire ioc order doesn't have fee
}

func Test_GTCPartialExpire(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{0x01, buyer, "b-1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 100000000, 100000000, orderPkg.TimeInForce.GTC}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, ""}, false)
	msg2 := orderPkg.NewOrderMsg{0x01, seller, "s-1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 300000000, orderPkg.TimeInForce.GTC}
	keeper.AddOrder(orderPkg.OrderInfo{msg2, 42, 100, 42, 100, 0, ""}, false)

	require.Len(keeper.OrderChanges, 2)
	require.Len(keeper.OrderChangesMap, 2)
	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	// verify orderChange0 - Ack
	assert.Equal("b-1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	assert.Equal("", orderChange0.FeeAsset)
	assert.Equal(int64(0), orderChange0.Fee)
	// verify orderChange1 - Ack
	assert.Equal("s-1", orderChange1.Id)
	assert.Equal(orderPkg.Ack, orderChange1.Tpe)
	assert.Equal("", orderChange1.FeeAsset)
	assert.Equal(int64(0), orderChange1.Fee)

	trades := MatchAndAllocateAllForPublish(keeper, am, ctx)
	require.Len(trades, 1)
	trade0 := trades[0]
	assert.Equal("0-0", trade0.Id)
	assert.Equal("XYZ_BNB", trade0.Symbol)
	assert.Equal(int64(100000000), trade0.Price)
	assert.Equal(int64(100000000), trade0.Qty)
	assert.Equal("s-1", trade0.Sid)
	assert.Equal("b-1", trade0.Bid)
	assert.Equal(int64(50000), trade0.Sfee)
	assert.Equal("BNB", trade0.SfeeAsset)
	assert.Equal(int64(50000), trade0.Bfee)
	assert.Equal("BNB", trade0.BfeeAsset)

	require.Len(keeper.OrderChanges, 2) // for GTC order, fully fill is not derived from transfer (will be generated by trade)
	require.Len(keeper.OrderChangesMap, 2)

	// let the sell order expire
	breathTime := prepareExpire(int64(43))
	ExpireOrdersForPublish(keeper, am, ctx, breathTime)

	require.Len(keeper.OrderChanges, 3)
	orderChange2 := keeper.OrderChanges[2]
	assert.Equal("s-1", orderChange2.Id)
	assert.Equal(orderPkg.Expired, orderChange2.Tpe)
	assert.Equal("", orderChange2.FeeAsset)
	assert.Equal(int64(0), orderChange2.Fee) // partially filled expire ioc order doesn't have fee
}

func Test_OneBuyVsTwoSell(t *testing.T) {
	assert, require := setupKeeperTest(t)

	msg := orderPkg.NewOrderMsg{0x01, buyer, "b-1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.BUY, 100000000, 300000000, orderPkg.TimeInForce.GTC}
	keeper.AddOrder(orderPkg.OrderInfo{msg, 42, 100, 42, 100, 0, ""}, false)
	msg2 := orderPkg.NewOrderMsg{0x01, seller, "s-1", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 100000000, orderPkg.TimeInForce.GTC}
	keeper.AddOrder(orderPkg.OrderInfo{msg2, 42, 100, 42, 100, 0, ""}, false)
	msg3 := orderPkg.NewOrderMsg{0x01, seller, "s-2", "XYZ_BNB", orderPkg.OrderType.LIMIT, orderPkg.Side.SELL, 100000000, 200000000, orderPkg.TimeInForce.GTC}
	keeper.AddOrder(orderPkg.OrderInfo{msg3, 42, 100, 42, 100, 0, ""}, false)

	require.Len(keeper.OrderChanges, 3)
	require.Len(keeper.OrderChangesMap, 3)
	orderChange0 := keeper.OrderChanges[0]
	orderChange1 := keeper.OrderChanges[1]
	orderChange2 := keeper.OrderChanges[2]
	// verify orderChange0 - Ack
	assert.Equal("b-1", orderChange0.Id)
	assert.Equal(orderPkg.Ack, orderChange0.Tpe)
	assert.Equal("", orderChange0.FeeAsset)
	assert.Equal(int64(0), orderChange0.Fee)
	// verify orderChange1 - Ack
	assert.Equal("s-1", orderChange1.Id)
	assert.Equal(orderPkg.Ack, orderChange1.Tpe)
	assert.Equal("", orderChange1.FeeAsset)
	assert.Equal(int64(0), orderChange1.Fee)
	// verify orderChange2 - Ack
	assert.Equal("s-2", orderChange2.Id)
	assert.Equal(orderPkg.Ack, orderChange2.Tpe)
	assert.Equal("", orderChange2.FeeAsset)
	assert.Equal(int64(0), orderChange2.Fee)

	trades := MatchAndAllocateAllForPublish(keeper, am, ctx)
	require.Len(trades, 2)
	trade0 := trades[0]
	assert.Equal("0-0", trade0.Id)
	assert.Equal("XYZ_BNB", trade0.Symbol)
	assert.Equal(int64(100000000), trade0.Price)
	assert.Equal(int64(100000000), trade0.Qty)
	assert.Equal("s-1", trade0.Sid)
	assert.Equal("b-1", trade0.Bid)
	assert.Equal(int64(50000), trade0.Sfee)
	assert.Equal("BNB", trade0.SfeeAsset)
	assert.Equal(int64(50000), trade0.Bfee)
	assert.Equal("BNB", trade0.BfeeAsset)
	trade1 := trades[1]
	assert.Equal("0-1", trade1.Id)
	assert.Equal("XYZ_BNB", trade1.Symbol)
	assert.Equal(int64(100000000), trade1.Price)
	assert.Equal(int64(200000000), trade1.Qty)
	assert.Equal("s-2", trade1.Sid)
	assert.Equal("b-1", trade1.Bid)
	assert.Equal(int64(100000), trade1.Sfee)
	assert.Equal("BNB", trade1.SfeeAsset)
	assert.Equal(int64(100000), trade1.Bfee)
	assert.Equal("BNB", trade1.BfeeAsset)
}

func prepareExpire(height int64) int64 {
	breathTime, _ := time.Parse(time.RFC3339, "2018-01-02T00:00:01Z")
	keeper.MarkBreatheBlock(ctx, height, breathTime.Unix())
	return breathTime.AddDate(0, 0, 3).Unix()
}
