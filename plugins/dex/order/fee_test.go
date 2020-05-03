package order

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/matcheng"
	dextype "github.com/binance-chain/node/plugins/dex/types"
)

func NewTestFeeConfig() FeeConfig {
	feeConfig := NewFeeConfig()
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

func TestFeeManager_calcTradeFeeForSingleTransfer(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.GlobalKeeper.FeeManager.UpdateConfig(NewTestFeeConfig())
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BNB", 1e7))
	keeper.AddEngine(dextype.NewTradingPair("BNB", "XYZ-111", 1e7))
	_, acc := testutils.NewAccount(ctx, am, 0)
	tran := Transfer{
		inAsset:  "ABC-000",
		in:       1000,
		outAsset: "BNB",
		out:      100,
	}
	// no enough bnb or native fee rounding to 0
	fee := keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"ABC-000", 1}}, fee.Tokens)
	_, acc = testutils.NewAccount(ctx, am, 100)
	fee = keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"ABC-000", 1}}, fee.Tokens)

	tran = Transfer{
		inAsset:  "ABC-000",
		in:       1000000,
		outAsset: "BNB",
		out:      10000,
	}
	_, acc = testutils.NewAccount(ctx, am, 1)
	fee = keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"ABC-000", 1000}}, fee.Tokens)
	_, acc = testutils.NewAccount(ctx, am, 100)
	fee = keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"BNB", 5}}, fee.Tokens)

	tran = Transfer{
		inAsset:  "BNB",
		in:       100,
		outAsset: "ABC-000",
		out:      1000,
	}
	_, acc = testutils.NewAccount(ctx, am, 100)
	fee = keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"BNB", 0}}, fee.Tokens)

	tran = Transfer{
		inAsset:  "BNB",
		in:       10000,
		outAsset: "ABC-000",
		out:      100000,
	}
	fee = keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"BNB", 5}}, fee.Tokens)

	tran = Transfer{
		inAsset:  "ABC-000",
		in:       100000,
		outAsset: "XYZ-111",
		out:      100000,
	}
	acc.SetCoins(sdk.Coins{{"ABC-000", 1000000}, {"BNB", 100}})
	fee = keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"BNB", 5}}, fee.Tokens)
	tran = Transfer{
		inAsset:  "XYZ-111",
		in:       100000,
		outAsset: "ABC-000",
		out:      100000,
	}
	acc.SetCoins(sdk.Coins{{"XYZ-111", 1000000}, {"BNB", 1000}})
	fee = keeper.GlobalKeeper.FeeManager.calcTradeFeeForSingleTransfer(acc.GetCoins(), &tran, keeper.engines)
	require.Equal(t, sdk.Coins{{"BNB", 500}}, fee.Tokens)
}

func TestFeeManager_CalcTradesFee(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.GlobalKeeper.FeeManager.UpdateConfig(NewTestFeeConfig())
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BNB", 1e7))
	keeper.AddEngine(dextype.NewTradingPair("XYZ-111", "BNB", 2e7))
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BTC", 1e4))
	keeper.AddEngine(dextype.NewTradingPair("XYZ-111", "BTC", 2e4))
	keeper.AddEngine(dextype.NewTradingPair("BNB", "BTC", 5e5))
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "XYZ-111", 6e7))

	tradeTransfers := TradeTransfers{
		{inAsset: "ABC-000", outAsset: "BNB", Oid: "1", in: 1e5, out: 2e4, Trade: &matcheng.Trade{}},
		{inAsset: "ABC-000", outAsset: "BTC", Oid: "2", in: 3e5, out: 4e1, Trade: &matcheng.Trade{}},
		{inAsset: "XYZ-111", outAsset: "BTC", Oid: "3", in: 2e6, out: 4e2, Trade: &matcheng.Trade{}},
		{inAsset: "XYZ-111", outAsset: "BNB", Oid: "4", in: 1e7, out: 2e6, Trade: &matcheng.Trade{}},
		{inAsset: "ABC-000", outAsset: "XYZ", Oid: "5", in: 8e6, out: 5e6, Trade: &matcheng.Trade{}},
		{inAsset: "BTC", outAsset: "BNB", Oid: "6", in: 1e8, out: 500e8, Trade: &matcheng.Trade{}},
		{inAsset: "BNB", outAsset: "BTC", Oid: "7", in: 300e8, out: 7e7, Trade: &matcheng.Trade{}},
		{inAsset: "BNB", outAsset: "ABC-000", Oid: "8", in: 5e8, out: 60e8, Trade: &matcheng.Trade{}},
		{inAsset: "ABC-000", outAsset: "BNB", Oid: "9", in: 7e6, out: 5e5, Trade: &matcheng.Trade{}},
		{inAsset: "ABC-000", outAsset: "BTC", Oid: "10", in: 6e5, out: 8e1, Trade: &matcheng.Trade{}},
	}
	_, acc := testutils.NewAccount(ctx, am, 0)
	_ = acc.SetCoins(sdk.Coins{
		{"ABC-000", 100e8},
		{"BNB", 15251400},
		{"BTC", 10e8},
		{"XYZ-000", 100e8},
	})
	fees := keeper.GlobalKeeper.FeeManager.CalcTradesFee(acc.GetCoins(), tradeTransfers, keeper.engines)
	require.Equal(t, "ABC-000:8000;BNB:15251305;BTC:100000;XYZ-111:2000", fees.String())
	require.Equal(t, "BNB:250000", tradeTransfers[0].Fee.String())
	require.Equal(t, "BNB:15000000", tradeTransfers[1].Fee.String())
	require.Equal(t, "BNB:10", tradeTransfers[2].Fee.String())
	require.Equal(t, "BNB:250", tradeTransfers[3].Fee.String())
	require.Equal(t, "BTC:100000", tradeTransfers[4].Fee.String())
	require.Equal(t, "BNB:1000", tradeTransfers[5].Fee.String())
	require.Equal(t, "BNB:15", tradeTransfers[6].Fee.String())
	require.Equal(t, "BNB:30", tradeTransfers[7].Fee.String())
	require.Equal(t, "ABC-000:8000", tradeTransfers[8].Fee.String())
	require.Equal(t, "XYZ-111:2000", tradeTransfers[9].Fee.String())
	require.Equal(t, sdk.Coins{
		{"ABC-000", 100e8},
		{"BNB", 15251400},
		{"BTC", 10e8},
		{"XYZ-000", 100e8},
	}, acc.GetCoins())
}

func TestFeeManager_CalcExpiresFee(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.GlobalKeeper.FeeManager.UpdateConfig(NewTestFeeConfig())
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BNB", 1e7))
	keeper.AddEngine(dextype.NewTradingPair("XYZ-111", "BNB", 2e7))
	keeper.AddEngine(dextype.NewTradingPair("BNB", "BTC", 5e5))

	// in BNB
	expireTransfers := ExpireTransfers{
		{inAsset: "ABC-000", Symbol: "ABC-000_BNB", Oid: "1"},
		{inAsset: "ABC-000", Symbol: "ABC-000_BTC", Oid: "2"},
		{inAsset: "XYZ-111", Symbol: "XYZ-111_BTC", Oid: "3"},
		{inAsset: "XYZ-111", Symbol: "XYZ-111_BNB", Oid: "4"},
		{inAsset: "ABC-000", Symbol: "ABC-000_XYZ-111", Oid: "5"},
		{inAsset: "BTC", Symbol: "BNB_BTC", Oid: "6"},
		{inAsset: "BNB", Symbol: "BNB_BTC", Oid: "7"},
		{inAsset: "BNB", Symbol: "ABC-000_BNB", Oid: "8"},
		{inAsset: "ABC-000", Symbol: "ABC-000_BNB", Oid: "9"},
		{inAsset: "ABC-000", Symbol: "ABC-000_BTC", Oid: "10"},
	}
	_, acc := testutils.NewAccount(ctx, am, 0)
	_ = acc.SetCoins(sdk.Coins{
		{"ABC-000", 100e8},
		{"BNB", 120000},
		{"BTC", 10e8},
		{"XYZ-111", 800000},
	})
	fees := keeper.GlobalKeeper.FeeManager.CalcExpiresFee(acc.GetCoins(), eventFullyExpire, expireTransfers, keeper.engines, nil)
	require.Equal(t, "ABC-000:1000000;BNB:120000;BTC:500;XYZ-111:800000", fees.String())
	require.Equal(t, "BNB:20000", expireTransfers[0].Fee.String())
	require.Equal(t, "BNB:20000", expireTransfers[1].Fee.String())
	require.Equal(t, "BNB:20000", expireTransfers[2].Fee.String())
	require.Equal(t, "BNB:20000", expireTransfers[3].Fee.String())
	require.Equal(t, "BNB:20000", expireTransfers[4].Fee.String())
	require.Equal(t, "BNB:20000", expireTransfers[5].Fee.String())
	require.Equal(t, "ABC-000:1000000", expireTransfers[6].Fee.String())
	require.Equal(t, "BTC:500", expireTransfers[7].Fee.String())
	require.Equal(t, "XYZ-111:500000", expireTransfers[8].Fee.String())
	require.Equal(t, "XYZ-111:300000", expireTransfers[9].Fee.String())
	require.Equal(t, sdk.Coins{
		{"ABC-000", 100e8},
		{"BNB", 120000},
		{"BTC", 10e8},
		{"XYZ-111", 800000},
	}, acc.GetCoins())
}

func TestFeeManager_CalcTradeFee(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.GlobalKeeper.FeeManager.UpdateConfig(NewTestFeeConfig())
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BNB", 1e7))
	// BNB
	_, acc := testutils.NewAccount(ctx, am, 0)
	// the tradeIn amount is large enough to make the fee > 0
	tradeIn := sdk.NewCoin(types.NativeTokenSymbol, 100e8)
	fee := keeper.GlobalKeeper.FeeManager.CalcTradeFee(acc.GetCoins(), tradeIn, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 5e6)}, fee.Tokens)
	// small tradeIn amount
	tradeIn = sdk.NewCoin(types.NativeTokenSymbol, 100)
	fee = keeper.GlobalKeeper.FeeManager.CalcTradeFee(acc.GetCoins(), tradeIn, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 0)}, fee.Tokens)

	// !BNB
	_, acc = testutils.NewAccount(ctx, am, 100)
	// has enough bnb
	tradeIn = sdk.NewCoin("ABC-000", 1000e8)
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e8)})
	fee = keeper.GlobalKeeper.FeeManager.CalcTradeFee(acc.GetCoins(), tradeIn, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 5e6)}, fee.Tokens)
	// no enough bnb
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e6)})
	fee = keeper.GlobalKeeper.FeeManager.CalcTradeFee(acc.GetCoins(), tradeIn, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC-000", 1e8)}, fee.Tokens)

	// very high price to produce int64 overflow
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BNB", 1e16))
	// has enough bnb
	tradeIn = sdk.NewCoin("ABC-000", 1000e8)
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e16)})
	fee = keeper.GlobalKeeper.FeeManager.CalcTradeFee(acc.GetCoins(), tradeIn, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 5e15)}, fee.Tokens)
	// no enough bnb, fee is within int64
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e15)})
	fee = keeper.GlobalKeeper.FeeManager.CalcTradeFee(acc.GetCoins(), tradeIn, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC-000", 1e8)}, fee.Tokens)
	// no enough bnb, even the fee overflows
	tradeIn = sdk.NewCoin("ABC-000", 1e16)
	fee = keeper.GlobalKeeper.FeeManager.CalcTradeFee(acc.GetCoins(), tradeIn, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC-000", 1e13)}, fee.Tokens)
}

func TestFeeManager_CalcFixedFee(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.GlobalKeeper.FeeManager.UpdateConfig(NewTestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 1e4)
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BNB", 1e7))
	keeper.AddEngine(dextype.NewTradingPair("BNB", "BTC-000", 1e5))
	// in BNB
	// no enough BNB, but inAsset == BNB
	fee := keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, types.NativeTokenSymbol, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e4)}, fee.Tokens)
	// enough BNB
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 3e4)})
	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, types.NativeTokenSymbol, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2e4)}, fee.Tokens)

	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventIOCFullyExpire, types.NativeTokenSymbol, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e4)}, fee.Tokens)

	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyCancel, types.NativeTokenSymbol, keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2e4)}, fee.Tokens)

	// ABC-000_BNB, sell ABC-000
	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "ABC-000", keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2e4)}, fee.Tokens)

	// No enough native token, but enough ABC-000
	acc.SetCoins(sdk.Coins{{Denom: types.NativeTokenSymbol, Amount: 1e4}, {Denom: "ABC-000", Amount: 1e8}})
	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "ABC-000", keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC-000", 1e6)}, fee.Tokens)

	// No enough native token and ABC-000
	acc.SetCoins(sdk.Coins{{Denom: types.NativeTokenSymbol, Amount: 1e4}, {Denom: "ABC-000", Amount: 1e5}})
	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "ABC-000", keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC-000", 1e5)}, fee.Tokens)

	// BNB_BTC-000, sell BTC-000
	acc.SetCoins(sdk.Coins{{Denom: "BTC-000", Amount: 1e4}})
	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "BTC-000", keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("BTC-000", 1e2)}, fee.Tokens)

	// extreme prices
	keeper.AddEngine(dextype.NewTradingPair("ABC-000", "BNB", 1))
	keeper.AddEngine(dextype.NewTradingPair("BNB", "BTC-000", 1e16))
	acc.SetCoins(sdk.Coins{{Denom: "ABC-000", Amount: 1e16}, {Denom: "BTC-000", Amount: 1e16}})
	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "ABC-000", keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC-000", 1e13)}, fee.Tokens)
	fee = keeper.GlobalKeeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "BTC-000", keeper.engines)
	require.Equal(t, sdk.Coins{sdk.NewCoin("BTC-000", 1e13)}, fee.Tokens)
}
