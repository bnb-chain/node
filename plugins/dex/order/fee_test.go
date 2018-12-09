package order

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/types"
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

func TestFeeManager_CalcOrderFees(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.FeeManager.UpdateConfig(ctx, NewTestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 0)
	lastTradePrices := map[string]int64{
		"ABC_BNB": 1e7,
	}
	// BNB
	tradeIn := sdk.NewCoin(types.NativeTokenSymbol, 100e8)
	fee := keeper.FeeManager.CalcOrderFee(acc.GetCoins(), tradeIn, lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 5e6)}, fee.Tokens)

	// !BNB
	_, acc = testutils.NewAccount(ctx, am, 100)
	// has enough bnb
	tradeIn = sdk.NewCoin("ABC", 1000e8)
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e8)})
	fee = keeper.FeeManager.CalcOrderFee(acc.GetCoins(), tradeIn, lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 5e6)}, fee.Tokens)
	// no enough bnb
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e6)})
	fee = keeper.FeeManager.CalcOrderFee(acc.GetCoins(), tradeIn, lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC", 1e8)}, fee.Tokens)
}

func TestFeeManager_CalcFixedFee(t *testing.T) {
	ctx, am, keeper := setup()
	keeper.FeeManager.UpdateConfig(ctx, NewTestFeeConfig())
	_, acc := testutils.NewAccount(ctx, am, 1e4)
	lastTradePrices := map[string]int64{
		"ABC_BNB": 1e7,
		"BNB_BTC": 1e5,
	}
	// in BNB
	// no enough BNB, but inAsset == BNB
	fee := keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, types.NativeTokenSymbol, lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e4)}, fee.Tokens)
	// enough BNB
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 3e4)})
	fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, types.NativeTokenSymbol, lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2e4)}, fee.Tokens)

	fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventIOCFullyExpire, types.NativeTokenSymbol, lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e4)}, fee.Tokens)

	fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyCancel, types.NativeTokenSymbol, lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2e4)}, fee.Tokens)

	// ABC_BNB, sell ABC
	fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "ABC", lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2e4)}, fee.Tokens)

	// No enough native token, but enough ABC
	acc.SetCoins(sdk.Coins{{Denom: types.NativeTokenSymbol, Amount: 1e4}, {Denom: "ABC", Amount: 1e8}})
	fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "ABC", lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC", 1e6)}, fee.Tokens)

	// No enough native token and ABC
	acc.SetCoins(sdk.Coins{{Denom: types.NativeTokenSymbol, Amount: 1e4}, {Denom: "ABC", Amount: 1e5}})
	fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "ABC", lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin("ABC", 1e5)}, fee.Tokens)

	// BNB_BTC, sell BTC
	acc.SetCoins(sdk.Coins{{Denom: "BTC", Amount: 1e4}})
	fee = keeper.FeeManager.CalcFixedFee(acc.GetCoins(), eventFullyExpire, "BTC", lastTradePrices)
	require.Equal(t, sdk.Coins{sdk.NewCoin("BTC", 1e2)}, fee.Tokens)
}
